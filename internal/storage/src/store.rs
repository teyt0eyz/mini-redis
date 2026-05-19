use std::collections::HashMap;
use std::sync::{Mutex, OnceLock};

static STORE: OnceLock<Mutex<HashMap<String, String>>> = OnceLock::new();

fn get_store() -> &'static Mutex<HashMap<String, String>> {
    STORE.get_or_init(|| Mutex::new(HashMap::new()))
}

pub fn set(key: &str, value: &str) {
    let mut store = get_store().lock().unwrap();
    store.insert(key.to_string(), value.to_string());
}

pub fn get(key: &str) -> Option<String> {
    let store = get_store().lock().unwrap();
    store.get(key).cloned()
}

pub fn delete(key: &str) -> bool {
    let mut store = get_store().lock().unwrap();
    store.remove(key).is_some()
}

pub fn exists(key: &str) -> bool {
    let store = get_store().lock().unwrap();
    store.contains_key(key)
}
