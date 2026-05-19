const std = @import("std");

// ประกาศดึงฟังก์ชันจากไฟล์สแตติกไลบรารีของ Rust ที่เราทำไว้
extern fn save_command(command_ptr: [*:0]const u8) void;

// ฟังก์ชันที่สร้างเปิดให้ Go เข้ามาเรียกใช้ (C ABI เหมือนกัน)
export fn process_and_forward(msg_ptr: [*:0]const u8, msg_len: usize) void {
    // 1. พิมพ์ Log ฝั่ง Zig ตามโจทย์เช้า
    std.debug.print("[Zig Core] กำลังประมวลผลคำสั่งความยาว {d} bytes...\n", .{msg_len});

    // 2. ส่งต่อพอยน์เตอร์ข้อมูลนี้ไปให้ Rust ทำงานต่อตรงๆ
    save_command(msg_ptr);
}
