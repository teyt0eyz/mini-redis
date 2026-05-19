use std::ffi::CStr;
   use libc::c_char;

   #[unsafe(no_mangle)]
   pub extern "C" fn save_command(command_ptr: *const c_char) {
       if command_ptr.is_null() {
           eprintln!("[Rust Storage] Error: Received null pointer");
           return;
       }

       unsafe {
           if let Ok(command_str) = CStr::from_ptr(command_ptr).to_str() {
               // พิมพ์ Log ปลายทางตามโจทย์เช้า
               println!("[Rust Storage] บันทึกข้อมูลสำเร็จ: {}", command_str);
           }
       }
   }