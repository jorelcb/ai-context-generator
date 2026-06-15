package dto

import "testing"

func TestSkillsConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     *SkillsConfig
		wantErr bool
	}{
		{
			"happy path",
			&SkillsConfig{OutputPath: "/tmp/x", Target: "claude"},
			false,
		},
		{
			"missing output",
			&SkillsConfig{Target: "claude"},
			true,
		},
		{
			"invalid target",
			&SkillsConfig{OutputPath: "/tmp/x", Target: "vscode"},
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestSpecConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     *SpecConfig
		wantErr bool
	}{
		{"happy", &SpecConfig{ProjectName: "x", FromContextPath: "/c", OutputPath: "/o", FeatureID: "x"}, false},
		{"missing name", &SpecConfig{FromContextPath: "/c", OutputPath: "/o", FeatureID: "x"}, true},
		{"missing context path", &SpecConfig{ProjectName: "x", OutputPath: "/o", FeatureID: "x"}, true},
		{"missing output", &SpecConfig{ProjectName: "x", FromContextPath: "/c", FeatureID: "x"}, true},
		{"missing feature id", &SpecConfig{ProjectName: "x", FromContextPath: "/c", OutputPath: "/o"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
