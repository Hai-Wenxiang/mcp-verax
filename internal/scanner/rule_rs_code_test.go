package scanner

import "testing"

func TestRustFixtureFindings(t *testing.T) {
	res, err := Scan(fixtureDir("scanrust"), "scanrust-fixture")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"VX-SEC-003": 2, // dynamic program + interpreter shell (fixed "ls" stays quiet)
		"VX-SEC-002": 1, // format! path (literal out.txt stays quiet)
		"VX-SEC-008": 1, // computed URL (literal URL stays quiet)
		"VX-SEC-014": 2, // unsafe block + unsafe impl
		"VX-QUA-107": 2, // unwrap + expect in src/ (tests/x.rs unwrap skipped)
		"VX-PRV-303": 1, // println! embedding token
		"VX-QUA-105": 4, // Cargo: no license/repository/description + no LICENSE file
	}
	got := map[string]int{}
	for _, f := range res.Findings {
		got[f.RuleID+f.File]++
		got[f.RuleID]++
		if f.File != "src/main.rs" && f.File != "Cargo.toml" && f.File != "." {
			t.Errorf("unexpected file for %s: %s", f.RuleID, f.File)
		}
	}
	for id, n := range want {
		if got[id] != n {
			for _, f := range findingsBy(res, id) {
				t.Logf("%s", f.Reason())
			}
			t.Errorf("%s findings = %d, want %d", id, got[id], n)
		}
	}
	if got["VX-QUA-107"] > 0 {
		for _, f := range findingsBy(res, "VX-QUA-107") {
			if f.File == "tests/x.rs" {
				t.Errorf("tests/ unwrap must not fire: %+v", f)
			}
		}
	}
}

func TestRustCommandExecEdges(t *testing.T) {
	cases := []struct {
		src  string
		want int
	}{
		{`Command::new("ls").arg("-l").spawn();`, 0},
		{`Command::new(r#"c:\tool"#).spawn();`, 0},
		{`Command::new("sh");`, 1},
		{`Command::new(&prog);`, 1},
		{`// Command::new(prog)`, 0},
		{"Command::new\n\t(format!(\"{}\", x));", 1},
	}
	for _, c := range cases {
		fs := rsCommandExec.Check("src/main.rs", c.src)
		if len(fs) != c.want {
			t.Errorf("%q → %d findings, want %d", c.src, len(fs), c.want)
		}
	}
}

func TestRustFSWriteContainment(t *testing.T) {
	clean := `fs::write(path, b"x");`
	if fs := rsFSWrite.Check("src/main.rs", clean); len(fs) != 1 {
		t.Errorf("computed write should fire, got %d", len(fs))
	}
	contained := `if path.starts_with(root) { fs::write(path, b"x"); }`
	if fs := rsFSWrite.Check("src/main.rs", contained); len(fs) != 0 {
		t.Errorf("containment marker should quiet the rule, got %d", len(fs))
	}
	literal := `fs::write("out.txt", b"x");`
	if fs := rsFSWrite.Check("src/main.rs", literal); len(fs) != 0 {
		t.Errorf("literal destination should stay quiet, got %d", len(fs))
	}
}

func TestRustOutboundGates(t *testing.T) {
	body := "let r = client.get(&url).send();"
	if fs := rsOutbound.Check("src/main.rs", body); len(fs) != 0 {
		t.Errorf("no HTTP client evidence should stay quiet, got %d", len(fs))
	}
	withReqwest := "use reqwest::Client;\n" + body
	if fs := rsOutbound.Check("src/main.rs", withReqwest); len(fs) != 1 {
		t.Errorf("reqwest + computed URL should fire, got %d", len(fs))
	}
	allowed := withReqwest + "\nconst ALLOWED_HOSTS: &[&str] = &[\"a\"];"
	if fs := rsOutbound.Check("src/main.rs", allowed); len(fs) != 0 {
		t.Errorf("allowlist should quiet the rule, got %d", len(fs))
	}
}

func TestRustCargoMeta(t *testing.T) {
	minimal := []byte("[package]\nname = \"x\"\nversion = \"0.1.0\"\n")
	fs := rsCargoMeta.Check("Cargo.toml", minimal)
	if len(fs) != 3 {
		t.Errorf("minimal Cargo.toml should flag 3 fields, got %d", len(fs))
	}
	full := []byte("[package]\nname = \"x\"\nversion = \"0.1.0\"\nlicense = \"MIT\"\nrepository = \"https://example.invalid/x\"\ndescription = \"d\"\n")
	if fs := rsCargoMeta.Check("Cargo.toml", full); len(fs) != 0 {
		t.Errorf("complete Cargo.toml should stay quiet, got %d", len(fs))
	}
	if fs := rsCargoMeta.Check("Cargo.toml", []byte("[workspace]\nmembers = [\"a\"]\n")); len(fs) != 0 {
		t.Errorf("workspace root manifest should stay quiet, got %d", len(fs))
	}
}
