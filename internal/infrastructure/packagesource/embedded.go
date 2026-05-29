// Package packagesource agrupa los adapters concretos del port
// catalog.PackageSource (ver ADR-0010 §8 y D.0 spike).
//
// Cada adapter conoce un origen (filesystem embedded, directorio local,
// repo Git, registry HTTP) y sabe enumerar + fetchear PackageManifests
// de ese origen. La composición de varias sources se hace en una capa
// superior (catalog command de D.4 / D.7).
package packagesource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jorelcb/codify/internal/domain/catalog"
)

// EmbeddedSource implementa catalog.PackageSource sobre el filesystem
// embebido en el binario (templates_embed.go en repo root). Es la fuente
// "default" — ofrece offline-by-default access al catálogo built-in que
// shippea con codify.
//
// Compone los converters de D.1.b para producir manifests, decora cada
// uno con SourceRef + Version + SourceChecksum, y resuelve Fetch leyendo
// directo del embed.FS.
//
// Workflows quedan fuera de v0 (defer a sub-hito posterior): su
// complejidad multi-target — el mismo template se instala como Claude
// skill o Antigravity workflow según el flag --target — requiere emitir
// dos manifests por workflow (uno por target) o un mecanismo separado.
// Skills + hooks (los dos casos del catalog UX por ADR-0010 §4) están
// cubiertos.
// Compile-time guard: EmbeddedSource must satisfy the frozen
// catalog.PackageSource contract. If the interface or this adapter drift
// apart, the build fails here instead of at a distant call site. Every
// future source adapter (local-fs, git, http-registry) must add the same
// assertion.
var _ catalog.PackageSource = (*EmbeddedSource)(nil)

type EmbeddedSource struct {
	// fsys es el filesystem embebido (típicamente codify.TemplatesFS).
	// Inyectado para permitir tests con FS mock sin dependencia del
	// package root.
	fsys fs.FS
	// version se decora en cada manifest emitido. Convención: el binary
	// version (.version file en runtime via cli.Version). Tests pasan
	// valores arbitrarios.
	version string
}

// NewEmbeddedSource construye un EmbeddedSource para el filesystem y
// versión dados.
//
// fsys debe contener una raíz "templates/" — la convención del repo.
// version es la versión semver del binario que produjo el embed (no es
// validada acá; el caller la pasa).
//
// Los artefactos distribuibles (skills + hooks) son inglés-only por
// diseño: viven en rutas locale-free (templates/skills/, templates/hooks/),
// así que la source no necesita un parámetro de locale. La localización
// solo aplica a generación LLM (context/spec/idioms), que no pasa por acá.
func NewEmbeddedSource(fsys fs.FS, version string) *EmbeddedSource {
	return &EmbeddedSource{fsys: fsys, version: version}
}

// Kind identifica este source en mensajes de UI/diagnóstico ("[embedded]").
func (s *EmbeddedSource) Kind() string { return "embedded" }

// List devuelve todos los manifests que la EmbeddedSource conoce: skills
// (uno por template) + hooks (uno por bundle). Decorados con SourceRef
// + Version + SourceChecksum.
//
// El error path es defensivo: si una category no se puede cargar (falla
// inesperada del filesystem embebido), se devuelve un error encapsulando
// la falla. En la práctica el embedded FS no falla — pero la interfaz
// requiere context+error para uniformidad con sources remotos (Git,
// HTTP).
func (s *EmbeddedSource) List(ctx context.Context) ([]catalog.PackageManifest, error) {
	var all []catalog.PackageManifest

	// Skills: iterar todas las categories registradas.
	for i := range catalog.Categories {
		cat := &catalog.Categories[i]
		manifests := catalog.ManifestsFromSkillsCategory(cat)
		all = append(all, manifests...)
	}

	// Hooks: la category única "hooks".
	if len(catalog.HookCategories) > 0 {
		hookCat := &catalog.HookCategories[0]
		manifests := catalog.ManifestsFromHooksCategory(hookCat)
		all = append(all, manifests...)
	}

	// Decorar cada manifest con metadata propia del source.
	for i := range all {
		s.decorate(&all[i])
	}

	// Re-sort global por (Target, ID) para tener un orden estable
	// cross-categories. Determinismo importa para tests + lockfiles.
	sort.Slice(all, func(i, j int) bool {
		if all[i].Target != all[j].Target {
			return all[i].Target < all[j].Target
		}
		return all[i].ID < all[j].ID
	})

	return all, nil
}

// decorate rellena los campos que el converter dejó vacíos: Source,
// Version, SourceChecksum. Idempotente — sobrescribe valores previos.
func (s *EmbeddedSource) decorate(m *catalog.PackageManifest) {
	m.Source = catalog.SourceRef{Kind: "embedded", URI: ""}
	m.Version = s.version

	// Checksum determinista basado en la "source declaration" — los
	// campos del manifest que no cambian entre fetchs. Excluimos campos
	// volátiles (InstalledAt, etc.) que no aplican acá. Si el caller
	// quiere validación contra contenido, debe hashear PackageContent.
	m.SourceChecksum = computeManifestChecksum(*m)
}

// computeManifestChecksum produce un hash estable de los campos
// declarativos del manifest. Sirve para detectar cambios entre catalog
// refreshes ("¿este package cambió desde la última vez que lo vi?"),
// no drift detection del archivo instalado (eso vive en el lockfile,
// D.9).
//
// La fórmula es SHA-256 de un string canonical: ID + Target + Description
// + Triggers ordenados + AllowedTools ordenados. Suficiente para detectar
// cambios visibles al usuario sin requerir el contenido completo del
// template (el hash del contenido se calcula por separado al Fetch si
// hace falta).
func computeManifestChecksum(m catalog.PackageManifest) string {
	var sb strings.Builder
	sb.WriteString(m.ID)
	sb.WriteByte('\x00')
	sb.WriteString(string(m.Target))
	sb.WriteByte('\x00')
	sb.WriteString(m.Description)
	sb.WriteByte('\x00')
	if m.Claude != nil {
		// Triggers/AllowedTools se incluyen ordenados para no depender
		// del orden de declaración.
		writeSortedSlice(&sb, m.Claude.Triggers)
		sb.WriteByte('\x00')
		writeSortedSlice(&sb, m.Claude.AllowedTools)
		sb.WriteByte('\x00')
		if m.Claude.UserInvocable {
			sb.WriteString("user-invocable")
		}
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(sum[:])
}

// writeSortedSlice escribe los elementos del slice ordenados, separados
// por NUL bytes. Asegura que dos slices con mismos elementos en distinto
// orden produzcan el mismo hash.
func writeSortedSlice(sb *strings.Builder, items []string) {
	if len(items) == 0 {
		return
	}
	sorted := append([]string{}, items...)
	sort.Strings(sorted)
	for i, s := range sorted {
		if i > 0 {
			sb.WriteByte('\x01')
		}
		sb.WriteString(s)
	}
}

// Fetch resuelve el contenido instalable de un manifest. La forma del
// PackageContent depende del Target:
//
//   - TargetClaudeSkill: Files["SKILL.md"] = frontmatter + template body.
//     SettingsFragment vacío.
//   - TargetClaudeHook: Files contiene los scripts del bundle (lint.sh,
//     etc.). SettingsFragment lleva el JSON merge (hooks.json).
//
// Otros targets aún no se manejan acá (se rechazan con error explícito).
func (s *EmbeddedSource) Fetch(ctx context.Context, m catalog.PackageManifest) (catalog.PackageContent, error) {
	switch m.Target {
	case catalog.TargetClaudeSkill:
		return s.fetchSkill(m)
	case catalog.TargetClaudeHook:
		return s.fetchHook(m)
	default:
		return catalog.PackageContent{}, fmt.Errorf("embedded source: target %q not supported in v0 (skills + hooks only)", m.Target)
	}
}

// fetchSkill arma el SKILL.md final para una skill: frontmatter generado
// por catalog.GenerateFrontmatter + template body leído del embedded FS.
//
// Reusa catalog.GenerateFrontmatter como única fuente de verdad del
// frontmatter de skills, compartida con el resto del catálogo.
func (s *EmbeddedSource) fetchSkill(m catalog.PackageManifest) (catalog.PackageContent, error) {
	guideName, body, err := s.skillTemplate(m)
	if err != nil {
		return catalog.PackageContent{}, err
	}

	// Frontmatter via catalog.GenerateFrontmatter — única fuente de verdad
	// del frontmatter de skills, compartida con el resto del catálogo.
	frontmatter := catalog.GenerateFrontmatter(guideName, "claude")
	content := frontmatter + "\n" + string(body)

	return catalog.PackageContent{
		Files: map[string][]byte{
			"SKILL.md": []byte(content),
		},
	}, nil
}

// skillTemplate resuelve el (guideName, rawBody) del template de un skill
// desde su manifest. Reusado por fetchSkill (static, le agrega frontmatter)
// y por PersonalizingSource (lo pasa al LLM como guide). El body es el
// template crudo, sin frontmatter.
func (s *EmbeddedSource) skillTemplate(m catalog.PackageManifest) (guideName string, body []byte, err error) {
	// Localizar el archivo: re-iterar las categories y matchear por guide
	// name. O(N) sobre el catálogo total, aceptable (~30 items).
	guideName, templateDir, err := s.locateSkillTemplate(m.ID)
	if err != nil {
		return "", nil, err
	}
	// Convención del repo: skills bajo templates/skills/<TemplateDir>/
	// (locale-free — inglés-only por diseño).
	templatePath := path.Join("templates", "skills", templateDir, guideName+".template")
	body, err = fs.ReadFile(s.fsys, templatePath)
	if err != nil {
		return "", nil, fmt.Errorf("read skill template %s: %w", templatePath, err)
	}
	return guideName, body, nil
}

// locateSkillTemplate encuentra el (guide_name, template_dir) de un
// skill identificado por su PackageManifest.ID. El ID es el guide name
// con underscores convertidos a guiones (convención de D.1.b
// normalizeID), así que la inversa es convertir guiones a underscores
// — pero hay que validar que el guide existe y bajo qué TemplateDir.
func (s *EmbeddedSource) locateSkillTemplate(id string) (guideName, templateDir string, err error) {
	guideName = strings.ReplaceAll(id, "-", "_")
	for _, cat := range catalog.Categories {
		for _, opt := range cat.Options {
			for tmplFile, tmplGuide := range opt.TemplateMapping {
				if tmplGuide == guideName {
					// tmplFile es típicamente "<guide>.template"; el dir
					// del template viene del SkillOption.TemplateDir.
					_ = tmplFile
					return guideName, opt.TemplateDir, nil
				}
			}
		}
	}
	return "", "", fmt.Errorf("skill %q not found in embedded catalog", id)
}

// fetchHook lee el bundle de hooks: scripts (.sh y otros) + el fragmento
// hooks.json que se merge en settings.json.
//
// Convención: el bundle vive en templates/hooks/<id>/ (locale-free). Todos
// los archivos del directorio se incluyen en Files; hooks.json (si
// existe) se separa hacia SettingsFragment para que el ClaudeInstaller
// haga el merge en lugar de copiarlo como archivo plano.
func (s *EmbeddedSource) fetchHook(m catalog.PackageManifest) (catalog.PackageContent, error) {
	bundleDir := path.Join("templates", "hooks", m.ID)
	entries, err := fs.ReadDir(s.fsys, bundleDir)
	if err != nil {
		return catalog.PackageContent{}, fmt.Errorf("read hook bundle %s: %w", bundleDir, err)
	}

	out := catalog.PackageContent{
		Files: make(map[string][]byte),
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// v0 no soporta sub-directories en hook bundles. Si surgen,
			// agregar walk recursivo.
			continue
		}
		name := entry.Name()
		filePath := path.Join(bundleDir, name)
		data, err := fs.ReadFile(s.fsys, filePath)
		if err != nil {
			return catalog.PackageContent{}, fmt.Errorf("read hook file %s: %w", filePath, err)
		}
		// Convención: archivos llamados "hooks.json" o "settings.json"
		// son fragmentos de merge para settings.json del usuario, no
		// scripts a copiar. Se separan a SettingsFragment.
		if filepath.Base(name) == "hooks.json" || filepath.Base(name) == "settings.json" {
			out.SettingsFragment = data
			continue
		}
		out.Files[name] = data
	}
	return out, nil
}
