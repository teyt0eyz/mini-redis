use std::time::{Duration, Instant};

pub struct Item {
    pub value: String,
    pub expires_at: Option<Instant>,
}

impl Item {
    pub fn new(value: String) -> Self {
        Self { value, expires_at: None }
    }

    pub fn with_ttl(value: String, secs: u64) -> Self {
        Self {
            value,
            expires_at: Some(Instant::now() + Duration::from_secs(secs)),
        }
    }

    pub fn is_expired(&self) -> bool {
        self.expires_at.map_or(false, |exp| Instant::now() > exp)
    }

    pub fn ttl_secs(&self) -> i64 {
        match self.expires_at {
            Some(exp) => {
                let now = Instant::now();
                if now > exp { -2 } else { (exp - now).as_secs() as i64 }
            }
            None => -1,
        }
    }
}
