#[test]
fn t() {
    // unwraps here are conventional and must not fire VX-QUA-107
    let x: Option<u8> = None;
    let _ = x.unwrap();
}
