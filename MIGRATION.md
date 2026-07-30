# Estado del repositorio y pasos pendientes de cierre

**Este repositorio está archivado (read-only) desde el 2026-07-30.** Contiene `codify` v1.x–v4.x,
el CLI y servidor MCP escrito en Go. Se llamaba `jorelcb/codify` hasta esa fecha.

El proyecto continúa como una **reescritura en Rust** en **[jorelcb/codify](https://github.com/jorelcb/codify)**,
que heredó el nombre. No es un refactor: es un producto sucesor, con otra arquitectura y otra
línea de versiones.

## Por qué se separaron

Ambos proyectos convivían en un mismo repositorio con **líneas de versión incompatibles**: este
publica `v4.x` y se instala vía Homebrew, mientras el sucesor arranca en `0.y.z` y estará
incompleto durante un tiempo. Un solo esquema de tags no puede servir a los dos sin romperle la
actualización a alguien.

## Qué sigue funcionando hoy

- **`brew install jorelcb/tap/codify`** instala el binario Go **v4.0.0** de este repositorio.
  La fórmula del tap ya apunta a los assets bajo `codify-og`, así que **no depende** del
  redirect de GitHub (que se desactivó al reutilizarse el nombre `codify`).
- Las **releases y sus assets** siguen descargables: un repositorio archivado permanece legible.
- El repositorio se puede clonar y consultar con normalidad.

---

# Pasos pendientes para el cierre definitivo

Ninguno es urgente. Se documentan para que la decisión no se pierda.

## 1. Deuda: `go install` no funciona con el module path nuevo

**Problema.** El módulo se renombró a `github.com/jorelcb/codify-og` (ver #44), pero los tags
`v1.0.0`–`v4.0.0` **ya publicados** conservan el path viejo en su `go.mod`. Por tanto:

```bash
go install github.com/jorelcb/codify-og/cmd/codify@v4.0.0   # falla: mismatch de module path
go install github.com/jorelcb/codify-og/cmd/codify@latest   # falla por lo mismo
```

Las instrucciones de `go install` del README quedan, en la práctica, inoperativas. El camino de
instalación soportado es **Homebrew** o descargar el asset del release.

**Cómo resolverlo** (solo si se quiere recuperar `go install`):

```bash
# 1. Desarchivar (ver §3)
# 2. El rename del módulo ya está en main; basta con publicar un tag nuevo:
git tag -a v4.0.1 -m "chore: republish with module path github.com/jorelcb/codify-og"
git push origin v4.0.1
# 3. goreleaser publica la release y actualiza la fórmula del tap automáticamente
# 4. Re-archivar
```

> ⚠️ **Publicar un tag es una acción pública e irreversible.** Requiere decisión explícita del
> responsable del proyecto. No se hizo durante la migración precisamente por eso.

**Alternativa sin tag**: editar el README para retirar las instrucciones de `go install` y dejar
Homebrew y los assets como únicos caminos documentados.

## 2. Transición de la fórmula de Homebrew

**Situación actual.** En `jorelcb/homebrew-tap`, la fórmula `Formula/codify.rb` se llama
`codify` y sirve **el binario Go de este repositorio**. El nombre de la fórmula es independiente
del nombre del repositorio, por eso siguió funcionando tras el rename.

**El conflicto llega cuando el sucesor empiece a distribuirse**, porque querrá el nombre
`codify`. Hay que decidir antes de publicar la primera versión distribuible de NG.

Un matiz que condiciona la decisión: **el sucesor es en su fase 1 una app de escritorio (Tauri)**,
no un CLI. En Homebrew eso normalmente es un **cask** (`.app`/`.dmg`), no una fórmula. Una piel
CLI existe en el plan, pero llegará después.

### Opciones

| Opción | Cómo queda | Ventaja | Costo |
|---|---|---|---|
| **A — Cask para NG, la fórmula no se toca** | `brew install --cask codify` (app NG) · `brew install jorelcb/tap/codify` (CLI Go) | Cero ruptura para quien ya tiene el CLI | Un mismo tap con fórmula y cask llamados `codify`: Homebrew avisa de ambigüedad |
| **B — NG toma `codify`, el legacy pasa a `codify-og`** | Se renombra `Formula/codify.rb` → `Formula/codify-og.rb` (y la clase `Codify` → `CodifyOg`) | Nombres limpios y sin ambigüedad | Las instalaciones existentes quedan apuntando a una fórmula que ya no existe con ese nombre; hay que comunicar la migración |
| **C — Versionar el legacy (convención Homebrew)** | `Formula/codify@4.rb` para el CLI Go, `codify` para NG | Es el patrón idiomático de Homebrew para versiones antiguas; convive sin ambigüedad | Requiere que quien quiera seguir en v4 reinstale con el nombre versionado |

**Recomendación**: **C** si NG llega a tener piel CLI (es la convención de Homebrew para esto),
o **A** mientras NG sea solo app de escritorio, que es lo que menos fricción genera a corto plazo.

### Pasos, sea cual sea la opción

1. Decidir el nombre de la fórmula/cask del legacy **antes** de publicar NG.
2. Renombrar o duplicar el archivo en `jorelcb/homebrew-tap` (recordar renombrar también la
   clase Ruby: debe coincidir con el nombre del archivo).
3. Si se renombra: si alguna vez se vuelve a publicar desde aquí, actualizar `brews.name` en
   [`.goreleaser.yml`](./.goreleaser.yml) para que no regenere la fórmula con el nombre viejo.
4. Actualizar el README del tap con ambos nombres y una nota de migración del estilo:
   `brew uninstall codify && brew install jorelcb/tap/<nombre-nuevo>`.
5. Verificar con una instalación limpia antes de anunciarlo.

## 3. Cómo desarchivar (si hace falta ejecutar algo de lo anterior)

Un repositorio archivado no admite commits, tags, issues ni ejecuciones de workflows.

```bash
gh api -X PATCH repos/jorelcb/codify-og -f archived=false   # desarchivar
# ... hacer el trabajo ...
gh repo archive jorelcb/codify-og --yes                     # volver a archivar
```

## 4. Comunicación de fin de vida

Pendiente de decidir: si se anuncia una fecha de EOL para v4.x o se mantiene disponible
indefinidamente como está. Mientras no haya anuncio, la interpretación por defecto es que **el
binario v4.0.0 seguirá descargable y funcionando, pero sin correcciones ni nuevas versiones**.

---

## Referencia rápida

| | Este repositorio | Sucesor |
|---|---|---|
| Repo | `jorelcb/codify-og` (archivado) | [`jorelcb/codify`](https://github.com/jorelcb/codify) |
| Lenguaje | Go | Rust (core hexagonal + app Tauri) |
| Versión | v4.0.0 (final) | `0.1.0` (desarrollo temprano) |
| Instalación | `brew install jorelcb/tap/codify` | aún no distribuido |
| Módulo Go | `github.com/jorelcb/codify-og` | — |
