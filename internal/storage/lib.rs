use std::ffi::{CStr, CString};
use libc::c_char;

#[path = "src/store.rs"]
mod store;

#[unsafe(no_mangle)]
pub extern "C" fn store_set(key_ptr: *const c_char, val_ptr: *const c_char) {
    if key_ptr.is_null() || val_ptr.is_null() { return; }
    unsafe {
        let key = CStr::from_ptr(key_ptr).to_str().unwrap_or("");
        let val = CStr::from_ptr(val_ptr).to_str().unwrap_or("");
        store::set(key, val);
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn store_get(key_ptr: *const c_char) -> *mut c_char {
    if key_ptr.is_null() { return std::ptr::null_mut(); }
    unsafe {
        let key = CStr::from_ptr(key_ptr).to_str().unwrap_or("");
        match store::get(key) {
            Some(val) => CString::new(val).unwrap().into_raw(),
            None => std::ptr::null_mut(),
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn store_del(key_ptr: *const c_char) -> i32 {
    if key_ptr.is_null() { return 0; }
    unsafe {
        let key = CStr::from_ptr(key_ptr).to_str().unwrap_or("");
        if store::delete(key) { 1 } else { 0 }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn store_exists(key_ptr: *const c_char) -> i32 {
    if key_ptr.is_null() { return 0; }
    unsafe {
        let key = CStr::from_ptr(key_ptr).to_str().unwrap_or("");
        if store::exists(key) { 1 } else { 0 }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn store_free_string(ptr: *mut c_char) {
    if !ptr.is_null() {
        unsafe { drop(CString::from_raw(ptr)) };
    }
}
