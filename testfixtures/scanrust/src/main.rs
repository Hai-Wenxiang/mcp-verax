use std::process::Command;
use std::fs;

fn main() {
    // dynamic program choice from input
    let prog = std::env::args().nth(1).unwrap_or_default();
    Command::new(prog).spawn();

    // interpreter shell: string becomes script
    Command::new("sh");

    // safe: fixed binary, arg-vector spawn
    Command::new("ls").arg("-l").spawn();

    // computed write path
    let name = std::env::args().nth(2).unwrap_or_default();
    fs::write(format!("/tmp/{}.txt", name), b"x");

    // safe: literal destination
    fs::write("out.txt", b"x");

    // outbound to computed URL
    let url = std::env::var("TARGET_URL").unwrap_or_default();
    let client = reqwest::blocking::Client::new();
    let _r = client.get(&url).send();
    let _s = client.get("https://example.invalid/health").send();

    // unsafe surfaces
    unsafe {
        let p = &mut 0u8;
        *p = 1;
    }
    unsafe impl Send for Wrapper {}

    // panic surfaces
    let m: Option<u8> = None;
    let _n = m.unwrap();
    let _e = m.expect("db row");

    // credentials to logs
    let token = String::new();
    println!("auth token: {}", token);
}
