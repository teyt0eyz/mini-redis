use std::collections::HashMap;
use std::sync::{Mutex, OnceLock};
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Instant;
use crate::item::Item;

static STORE: OnceLock<Mutex<HashMap<String, Item>>> = OnceLock::new();
static EXPIRED_COUNT: AtomicU64 = AtomicU64::new(0);
static MAX_KEYS: AtomicU64 = AtomicU64::new(0);

fn get_store() -> &'static Mutex<HashMap<String, Item>> {
    STORE.get_or_init(|| Mutex::new(HashMap::new()))
}

fn evict_lru(map: &mut HashMap<String, Item>) {
    if let Some(key) = map
        .iter()
        .min_by_key(|(_, item)| item.last_accessed)
        .map(|(k, _)| k.clone())
    {
        map.remove(&key);
    }
}

pub fn set_max_keys(n: u64) {
    MAX_KEYS.store(n, Ordering::Relaxed);
}

pub fn set(key: &str, value: &str) {
    let mut map = get_store().lock().unwrap();
    let max = MAX_KEYS.load(Ordering::Relaxed);
    if max > 0 && map.len() >= max as usize && !map.contains_key(key) {
        evict_lru(&mut map);
    }
    map.insert(key.to_string(), Item::new(value.to_string()));
}

pub fn set_ex(key: &str, value: &str, secs: u64) {
    let mut map = get_store().lock().unwrap();
    let max = MAX_KEYS.load(Ordering::Relaxed);
    if max > 0 && map.len() >= max as usize && !map.contains_key(key) {
        evict_lru(&mut map);
    }
    map.insert(key.to_string(), Item::with_ttl(value.to_string(), secs));
}

pub fn get(key: &str) -> Option<String> {
    let mut map = get_store().lock().unwrap();
    let expired = map.get(key).map_or(false, |item| item.is_expired());
    if expired {
        map.remove(key);
        EXPIRED_COUNT.fetch_add(1, Ordering::Relaxed);
        return None;
    }
    if let Some(item) = map.get_mut(key) {
        item.last_accessed = Instant::now();
        Some(item.value.clone())
    } else {
        None
    }
}

pub fn delete(key: &str) -> bool {
    let mut map = get_store().lock().unwrap();
    map.remove(key).is_some()
}

pub fn exists(key: &str) -> bool {
    let map = get_store().lock().unwrap();
    match map.get(key) {
        Some(item) => !item.is_expired(),
        None => false,
    }
}

pub fn ttl(key: &str) -> i64 {
    let map = get_store().lock().unwrap();
    match map.get(key) {
        Some(item) => item.ttl_secs(),
        None => -2,
    }
}

pub fn cleanup_expired() {
    let mut map = get_store().lock().unwrap();
    let before = map.len();
    map.retain(|_, item| !item.is_expired());
    let removed = before - map.len();
    if removed > 0 {
        EXPIRED_COUNT.fetch_add(removed as u64, Ordering::Relaxed);
    }
}

pub fn expired_count() -> u64 {
    EXPIRED_COUNT.load(Ordering::Relaxed)
}

pub fn incr(key: &str) -> Result<i64, &'static str> {
    let mut map = get_store().lock().unwrap();
    let current: i64 = match map.get(key) {
        Some(item) if item.is_expired() => {
            map.remove(key);
            EXPIRED_COUNT.fetch_add(1, Ordering::Relaxed);
            0
        }
        Some(item) => item.value.parse::<i64>()
            .map_err(|_| "ERR value is not an integer or out of range")?,
        None => 0,
    };
    let new_val = current.checked_add(1)
        .ok_or("ERR increment or decrement would overflow")?;
    let max = MAX_KEYS.load(Ordering::Relaxed);
    if max > 0 && map.len() >= max as usize && !map.contains_key(key) {
        evict_lru(&mut map);
    }
    map.insert(key.to_string(), Item::new(new_val.to_string()));
    Ok(new_val)
}
