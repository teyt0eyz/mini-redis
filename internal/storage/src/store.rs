use std::collections::HashMap;
use std::sync::{Mutex, OnceLock};
use crate::item::Item;

static STORE: OnceLock<Mutex<HashMap<String, Item>>> = OnceLock::new();

fn get_store() -> &'static Mutex<HashMap<String, Item>> {
    STORE.get_or_init(|| Mutex::new(HashMap::new()))
}

pub fn set(key: &str, value: &str) {
    let mut store = get_store().lock().unwrap();
    store.insert(key.to_string(), Item::new(value.to_string()));
}

pub fn set_ex(key: &str, value: &str, secs: u64) {
    let mut store = get_store().lock().unwrap();
    store.insert(key.to_string(), Item::with_ttl(value.to_string(), secs));
}

pub fn get(key: &str) -> Option<String> {
    let store = get_store().lock().unwrap();
    match store.get(key) {
        Some(item) if !item.is_expired() => Some(item.value.clone()),
        _ => None,
    }
}

pub fn delete(key: &str) -> bool {
    let mut store = get_store().lock().unwrap();
    store.remove(key).is_some()
}

pub fn exists(key: &str) -> bool {
    let store = get_store().lock().unwrap();
    match store.get(key) {
        Some(item) => !item.is_expired(),
        None => false,
    }
}

pub fn ttl(key: &str) -> i64 {
    let store = get_store().lock().unwrap();
    match store.get(key) {
        Some(item) => item.ttl_secs(),
        None => -2,
    }
}

pub fn cleanup_expired() {
    let mut store = get_store().lock().unwrap();
    store.retain(|_, item| !item.is_expired());
}
