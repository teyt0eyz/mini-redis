const std = @import("std");

extern fn store_set(key: [*:0]const u8, val: [*:0]const u8) void;
extern fn store_get(key: [*:0]const u8) [*c]u8;
extern fn store_del(key: [*:0]const u8) i32;
extern fn store_exists(key: [*:0]const u8) i32;
extern fn store_free_string(ptr: [*c]u8) void;

export fn zig_set(key: [*:0]const u8, val: [*:0]const u8) void {
    store_set(key, val);
}

export fn zig_get(key: [*:0]const u8) [*c]u8 {
    return store_get(key);
}

export fn zig_del(key: [*:0]const u8) i32 {
    return store_del(key);
}

export fn zig_exists(key: [*:0]const u8) i32 {
    return store_exists(key);
}

export fn zig_free_string(ptr: [*c]u8) void {
    store_free_string(ptr);
}
