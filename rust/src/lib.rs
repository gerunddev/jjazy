use libc::c_char;
use serde::Serialize;
use std::ffi::{CStr, CString};
use std::path::Path;
use std::ptr;
use std::sync::Arc;

use jj_lib::config::StackedConfig;
use jj_lib::repo::ReadonlyRepo;
use jj_lib::settings::UserSettings;
use jj_lib::workspace::{default_working_copy_factories, Workspace};

/// Opaque handle to a jj repository
pub struct RepoHandle {
    repo: Arc<ReadonlyRepo>,
}

/// Result structure for FFI calls
#[repr(C)]
pub struct JjResult {
    /// JSON data on success, NULL on error
    data: *mut c_char,
    /// Error message on failure, NULL on success
    error: *mut c_char,
}

/// Branch information for serialization
#[derive(Serialize)]
struct BranchInfo {
    name: String,
    is_local: bool,
}

impl JjResult {
    fn success(data: String) -> Self {
        JjResult {
            data: CString::new(data).unwrap().into_raw(),
            error: ptr::null_mut(),
        }
    }

    fn error(msg: String) -> Self {
        JjResult {
            data: ptr::null_mut(),
            error: CString::new(msg).unwrap().into_raw(),
        }
    }
}

fn create_user_settings() -> Result<UserSettings, String> {
    // Create minimal user settings with required defaults
    use jj_lib::config::{ConfigLayer, ConfigSource};
    use std::fs;

    let mut config = StackedConfig::empty();

    // Built-in defaults that match what jj expects
    let defaults = ConfigLayer::parse(
        ConfigSource::Default,
        r#"
        [user]
        name = "jayz"
        email = "jayz@localhost"

        [operation]
        hostname = "localhost"
        username = "jayz"

        [signing]
        behavior = "drop"
        backend = "gpg"

        [signing.backends.gpg]
        program = "gpg"
        allow-expired-keys = false

        [signing.backends.ssh]
        program = "ssh-keygen"
        allowed-signers = ""

        [signing.backends.gpgsm]
        program = "gpgsm"
        allow-expired-keys = false

        [ui]
        color = "auto"
        paginate = "auto"
        default-command = "log"
        conflict-marker-style = "diff"

        [merge]
        hunk-level = "line"
        same-change = "keep"

        [git]
        auto-local-bookmark = true
        abandon-unreachable-commits = true
        private-commits = "none()"
        executable-path = "git"
        write-change-id-header = false
        push-new-bookmarks = false
        sign-on-push = false

        [working-copy]
        eol-conversion = "none"

        [fsmonitor]
        backend = "none"
        "#,
    )
    .map_err(|e| e.to_string())?;

    config.add_layer(defaults);

    // Try to load user config from standard location
    if let Some(home) = std::env::var_os("HOME") {
        let config_path = Path::new(&home).join(".config/jj/config.toml");
        if config_path.exists() {
            if let Ok(content) = fs::read_to_string(&config_path) {
                if let Ok(user_config) = ConfigLayer::parse(ConfigSource::User, &content) {
                    config.add_layer(user_config);
                }
            }
        }
    }

    UserSettings::from_config(config).map_err(|e| e.to_string())
}

/// Open a jj repository at the given path
/// Returns NULL on error (check stderr)
#[no_mangle]
pub extern "C" fn jj_open_repo(path: *const c_char) -> *mut RepoHandle {
    let path_str = unsafe {
        if path.is_null() {
            eprintln!("jj_open_repo: null path");
            return ptr::null_mut();
        }
        match CStr::from_ptr(path).to_str() {
            Ok(s) => s,
            Err(e) => {
                eprintln!("jj_open_repo: invalid UTF-8: {}", e);
                return ptr::null_mut();
            }
        }
    };

    let path = Path::new(path_str);

    let settings = match create_user_settings() {
        Ok(s) => s,
        Err(e) => {
            eprintln!("jj_open_repo: failed to create settings: {}", e);
            return ptr::null_mut();
        }
    };

    let working_copy_factories = default_working_copy_factories();
    match Workspace::load(&settings, path, &Default::default(), &working_copy_factories) {
        Ok(workspace) => {
            match workspace.repo_loader().load_at_head() {
                Ok(repo) => {
                    let handle = Box::new(RepoHandle { repo });
                    Box::into_raw(handle)
                }
                Err(e) => {
                    eprintln!("jj_open_repo: failed to load repo at head: {:?}", e);
                    ptr::null_mut()
                }
            }
        }
        Err(e) => {
            eprintln!("jj_open_repo: failed to load workspace: {:?}", e);
            ptr::null_mut()
        }
    }
}

/// List branches in the repository
/// Returns JjResult with JSON array of branch names on success
#[no_mangle]
pub extern "C" fn jj_list_branches(handle: *mut RepoHandle) -> JjResult {
    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &*handle
    };

    let mut branches = Vec::new();

    // Get local branches (bookmarks in jj terminology) from the view
    for (name, _target) in handle.repo.view().local_bookmarks() {
        branches.push(BranchInfo {
            name: name.as_str().to_string(),
            is_local: true,
        });
    }

    match serde_json::to_string(&branches) {
        Ok(json) => JjResult::success(json),
        Err(e) => JjResult::error(format!("JSON serialization failed: {}", e)),
    }
}

/// Close a repository handle and free its memory
#[no_mangle]
pub extern "C" fn jj_close_repo(handle: *mut RepoHandle) {
    if !handle.is_null() {
        unsafe {
            drop(Box::from_raw(handle));
        }
    }
}

/// Free a JjResult's memory
#[no_mangle]
pub extern "C" fn jj_free_result(result: JjResult) {
    if !result.data.is_null() {
        unsafe {
            drop(CString::from_raw(result.data));
        }
    }
    if !result.error.is_null() {
        unsafe {
            drop(CString::from_raw(result.error));
        }
    }
}

/// Free a string allocated by Rust
#[no_mangle]
pub extern "C" fn jj_free_string(s: *mut c_char) {
    if !s.is_null() {
        unsafe {
            drop(CString::from_raw(s));
        }
    }
}
