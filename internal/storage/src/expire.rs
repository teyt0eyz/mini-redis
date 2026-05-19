use std::thread;
use std::time::Duration;

pub fn start_cleanup_loop() {
    thread::spawn(|| loop {
        thread::sleep(Duration::from_secs(1));
        crate::store::cleanup_expired();
    });
}
