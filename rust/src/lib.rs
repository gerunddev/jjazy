use libc::c_char;
use serde::Serialize;
use std::ffi::{CStr, CString};
use std::path::Path;
use std::ptr;
use std::sync::Arc;

use jj_lib::commit::Commit;
use jj_lib::config::StackedConfig;
use jj_lib::matchers::EverythingMatcher;
use jj_lib::merged_tree::MergedTree;
use jj_lib::object_id::ObjectId;
use jj_lib::repo::{ReadonlyRepo, Repo};
use jj_lib::settings::UserSettings;
use jj_lib::workspace::{default_working_copy_factories, Workspace};

/// Opaque handle to a jj repository
pub struct RepoHandle {
    repo: Arc<ReadonlyRepo>,
    current_workspace: String,
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

/// Workspace information for serialization
#[derive(Serialize)]
struct WorkspaceInfo {
    name: String,
    is_current: bool,
    commit_id: String,
}

/// File change information for serialization
#[derive(Serialize)]
struct FileChangeInfo {
    path: String,
    status: String, // "modified", "added", "deleted"
}

/// File contents for before/after comparison
#[derive(Serialize)]
struct FileContents {
    before: String,
    after: String,
    path: String,
}

/// Operation information for serialization
#[derive(Serialize)]
struct OperationInfo {
    id: String,
    description: String,
    timestamp: String,
    is_current: bool,
}

/// Revision information for serialization
#[derive(Serialize)]
struct RevisionInfo {
    id: String,
    change_id: String,
    description: String,
    author: String,
    timestamp: String,
    bookmarks: Vec<String>,
    git_head: bool,
    is_working_copy: bool,
    workspace_name: Option<String>,
    is_root: bool,
    parents: Vec<String>,
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
        name = "jjazy"
        email = "jjazy@localhost"

        [operation]
        hostname = "localhost"
        username = "jjazy"

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
            let workspace_name = workspace.workspace_name().as_str().to_string();
            match workspace.repo_loader().load_at_head() {
                Ok(repo) => {
                    let handle = Box::new(RepoHandle {
                        repo,
                        current_workspace: workspace_name,
                    });
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

/// List workspaces in the repository
/// Returns JjResult with JSON array of workspace info on success
#[no_mangle]
pub extern "C" fn jj_list_workspaces(handle: *mut RepoHandle) -> JjResult {
    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &*handle
    };

    let mut workspaces = Vec::new();

    // Get all workspaces from the view's working copy commit IDs
    for (workspace_id, commit_id) in handle.repo.view().wc_commit_ids() {
        let ws_name = workspace_id.as_str().to_string();
        workspaces.push(WorkspaceInfo {
            name: ws_name.clone(),
            is_current: ws_name == handle.current_workspace,
            commit_id: commit_id.hex(),
        });
    }

    // Sort workspaces by name for consistent ordering
    workspaces.sort_by(|a, b| a.name.cmp(&b.name));

    match serde_json::to_string(&workspaces) {
        Ok(json) => JjResult::success(json),
        Err(e) => JjResult::error(format!("JSON serialization failed: {}", e)),
    }
}

/// Get file changes in the current working copy
/// Returns JjResult with JSON array of file change info on success
#[no_mangle]
pub extern "C" fn jj_get_working_copy_changes(handle: *mut RepoHandle) -> JjResult {
    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &*handle
    };

    // Find the current workspace's working copy commit
    let wc_commit_id = match handle
        .repo
        .view()
        .wc_commit_ids()
        .iter()
        .find(|(ws_id, _)| ws_id.as_str() == handle.current_workspace)
    {
        Some((_, commit_id)) => commit_id.clone(),
        None => return JjResult::error("No working copy found for current workspace".to_string()),
    };

    // Get the working copy commit
    let wc_commit: Commit = match handle.repo.store().get_commit(&wc_commit_id) {
        Ok(commit) => commit,
        Err(e) => return JjResult::error(format!("Failed to get working copy commit: {}", e)),
    };

    // Get the parent commit(s) - use first parent for diff
    let parent_ids = wc_commit.parent_ids();
    if parent_ids.is_empty() {
        // Root commit - compare against empty tree
        return JjResult::success("[]".to_string());
    }

    let parent_commit: Commit = match handle.repo.store().get_commit(&parent_ids[0]) {
        Ok(commit) => commit,
        Err(e) => return JjResult::error(format!("Failed to get parent commit: {}", e)),
    };

    // Get trees for comparison
    let parent_tree: MergedTree = parent_commit.tree();
    let wc_tree: MergedTree = wc_commit.tree();

    // Collect file changes using diff_stream
    let mut changes = Vec::new();
    let matcher = EverythingMatcher;

    // Use diff_stream and collect synchronously
    let diff_stream = parent_tree.diff_stream(&wc_tree, &matcher);

    // Use pollster to block on the async stream
    pollster::block_on(async {
        use futures_util::StreamExt;
        futures_util::pin_mut!(diff_stream);
        while let Some(entry) = diff_stream.next().await {
            // entry is a TreeDiffEntry with path and values fields
            let diff_values = match entry.values {
                Ok(v) => v,
                Err(_) => continue,
            };

            let status = if diff_values.before.is_absent() && !diff_values.after.is_absent() {
                "added"
            } else if !diff_values.before.is_absent() && diff_values.after.is_absent() {
                "deleted"
            } else {
                "modified"
            };

            changes.push(FileChangeInfo {
                path: entry.path.as_internal_file_string().to_string(),
                status: status.to_string(),
            });
        }
    });

    match serde_json::to_string(&changes) {
        Ok(json) => JjResult::success(json),
        Err(e) => JjResult::error(format!("JSON serialization failed: {}", e)),
    }
}

/// List recent operations in the repository
/// Returns JjResult with JSON array of operation info on success
#[no_mangle]
pub extern "C" fn jj_list_operations(handle: *mut RepoHandle) -> JjResult {
    use jj_lib::operation::Operation;

    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &*handle
    };

    let mut operations = Vec::new();
    let op_store = handle.repo.op_store();
    let current_op = handle.repo.operation();
    let current_op_id = current_op.id().clone();

    // Walk through operations (current + parents)
    let mut to_visit = vec![current_op.clone()];
    let max_ops = 50; // Limit to recent operations

    while let Some(op) = to_visit.pop() {
        if operations.len() >= max_ops {
            break;
        }

        let op_id = op.id().hex();
        let is_current = *op.id() == current_op_id;
        let metadata = op.metadata();
        let description = metadata.description.clone();

        // Format timestamp
        let timestamp = format!("{}", metadata.time.start.timestamp.0);

        operations.push(OperationInfo {
            id: op_id[..12].to_string(), // Short ID
            description,
            timestamp,
            is_current,
        });

        // Add parent operations to visit
        for parent_id in op.parent_ids() {
            let parent_op_result = pollster::block_on(op_store.read_operation(parent_id));
            if let Ok(parent_op_data) = parent_op_result {
                to_visit.push(Operation::new(
                    op_store.clone(),
                    parent_id.clone(),
                    parent_op_data,
                ));
            }
        }
    }

    match serde_json::to_string(&operations) {
        Ok(json) => JjResult::success(json),
        Err(e) => JjResult::error(format!("JSON serialization failed: {}", e)),
    }
}

/// Get revision log for the repository
/// Returns JjResult with JSON array of revision info on success
#[no_mangle]
pub extern "C" fn jj_get_log(handle: *mut RepoHandle) -> JjResult {
    use std::collections::HashSet;

    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &*handle
    };

    // Build a map of workspace commit IDs to workspace names
    let mut workspace_commits: std::collections::HashMap<String, String> =
        std::collections::HashMap::new();
    for (ws_id, commit_id) in handle.repo.view().wc_commit_ids() {
        workspace_commits.insert(commit_id.hex(), ws_id.as_str().to_string());
    }

    // Get the root commit ID
    let root_commit_id = handle.repo.store().root_commit_id().hex();

    let mut revisions = Vec::new();
    let mut visited: HashSet<String> = HashSet::new();
    let max_revisions = 100;

    // Start from all working copy commits and walk up the graph
    let mut to_visit: Vec<jj_lib::commit::Commit> = Vec::new();
    for (_ws_id, commit_id) in handle.repo.view().wc_commit_ids() {
        if let Ok(commit) = handle.repo.store().get_commit(commit_id) {
            to_visit.push(commit);
        }
    }

    while let Some(commit) = to_visit.pop() {
        if revisions.len() >= max_revisions {
            break;
        }

        let commit_id_hex = commit.id().hex();
        if visited.contains(&commit_id_hex) {
            continue;
        }
        visited.insert(commit_id_hex.clone());

        let is_root = commit_id_hex == root_commit_id;
        // Use reverse_hex() for change IDs to get the base32-like encoding (e.g., "pmyysvqp")
        let change_id = commit.change_id().reverse_hex();
        let description = commit.description().to_string();

        // Get author info
        let signature = commit.author();
        let author = signature.email.clone();

        // Format timestamp as date + time
        let commit_ts_secs = signature.timestamp.timestamp.0 / 1000;
        let datetime = chrono::DateTime::from_timestamp(commit_ts_secs, 0)
            .map(|dt| dt.format("%Y-%m-%d %H:%M").to_string())
            .unwrap_or_else(|| "unknown".to_string());
        let timestamp = datetime;

        // Check if this is a working copy commit
        let workspace_name = workspace_commits.get(&commit_id_hex).cloned();
        let is_working_copy = workspace_name.is_some();

        // Get parent commit IDs
        let parents: Vec<String> = commit.parent_ids().iter().map(|id| id.hex()).collect();

        // Get bookmarks for this commit
        let mut bookmarks: Vec<String> = Vec::new();
        let mut git_head = false;

        for (name, target) in handle.repo.view().local_bookmarks() {
            if target.added_ids().any(|id| id == commit.id()) {
                bookmarks.push(name.as_str().to_string());
            }
        }

        // Check if this commit is at git HEAD
        let git_head_ref = handle.repo.view().git_head();
        if git_head_ref.added_ids().any(|id| id == commit.id()) {
            git_head = true;
        }

        revisions.push(RevisionInfo {
            id: commit_id_hex[..12].to_string(),
            change_id: change_id[..12].to_string(),
            description,
            author,
            timestamp,
            bookmarks,
            git_head,
            is_working_copy,
            workspace_name,
            is_root,
            parents: parents.iter().map(|p| p[..12].to_string()).collect(),
        });

        // Add parent commits to visit
        if !is_root {
            for parent_id in commit.parent_ids() {
                if !visited.contains(&parent_id.hex()) {
                    if let Ok(parent_commit) = handle.repo.store().get_commit(parent_id) {
                        to_visit.push(parent_commit);
                    }
                }
            }
        }
    }

    match serde_json::to_string(&revisions) {
        Ok(json) => JjResult::success(json),
        Err(e) => JjResult::error(format!("JSON serialization failed: {}", e)),
    }
}

/// Get diff for the working copy (changes from parent)
/// Returns JjResult with unified diff string on success
#[no_mangle]
pub extern "C" fn jj_get_diff(handle: *mut RepoHandle) -> JjResult {
    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &*handle
    };

    // Find the current workspace's working copy commit
    let wc_commit_id = match handle
        .repo
        .view()
        .wc_commit_ids()
        .iter()
        .find(|(ws_id, _)| ws_id.as_str() == handle.current_workspace)
    {
        Some((_, commit_id)) => commit_id.clone(),
        None => return JjResult::error("No working copy found for current workspace".to_string()),
    };

    // Get the working copy commit
    let wc_commit: Commit = match handle.repo.store().get_commit(&wc_commit_id) {
        Ok(commit) => commit,
        Err(e) => return JjResult::error(format!("Failed to get working copy commit: {}", e)),
    };

    // Get the parent commit(s)
    let parent_ids = wc_commit.parent_ids();
    if parent_ids.is_empty() {
        return JjResult::success("".to_string());
    }

    let parent_commit: Commit = match handle.repo.store().get_commit(&parent_ids[0]) {
        Ok(commit) => commit,
        Err(e) => return JjResult::error(format!("Failed to get parent commit: {}", e)),
    };

    // Get trees for comparison
    let parent_tree: MergedTree = parent_commit.tree();
    let wc_tree: MergedTree = wc_commit.tree();

    // Collect diff output
    let mut diff_output = String::new();
    let matcher = EverythingMatcher;

    let diff_stream = parent_tree.diff_stream(&wc_tree, &matcher);

    pollster::block_on(async {
        use futures_util::StreamExt;
        futures_util::pin_mut!(diff_stream);

        while let Some(entry) = diff_stream.next().await {
            let diff_values = match entry.values {
                Ok(v) => v,
                Err(_) => continue,
            };

            let path = entry.path.as_internal_file_string();

            // Determine the change type
            let before_is_file = !diff_values.before.is_absent();
            let after_is_file = !diff_values.after.is_absent();

            if before_is_file && after_is_file {
                // Modified file - get content diff
                diff_output.push_str(&format!("diff --git a/{} b/{}\n", path, path));
                diff_output.push_str(&format!("--- a/{}\n", path));
                diff_output.push_str(&format!("+++ b/{}\n", path));

                // Get file contents for diff
                let before_content = get_file_content(&handle.repo, &diff_values.before);
                let after_content = get_file_content(&handle.repo, &diff_values.after);

                // Generate line-based diff
                let diff_lines = generate_unified_diff(&before_content, &after_content);
                diff_output.push_str(&diff_lines);
            } else if !before_is_file && after_is_file {
                // Added file
                diff_output.push_str(&format!("diff --git a/{} b/{}\n", path, path));
                diff_output.push_str("new file\n");
                diff_output.push_str(&format!("--- /dev/null\n"));
                diff_output.push_str(&format!("+++ b/{}\n", path));

                let after_content = get_file_content(&handle.repo, &diff_values.after);
                for line in after_content.lines() {
                    diff_output.push_str(&format!("+{}\n", line));
                }
            } else if before_is_file && !after_is_file {
                // Deleted file
                diff_output.push_str(&format!("diff --git a/{} b/{}\n", path, path));
                diff_output.push_str("deleted file\n");
                diff_output.push_str(&format!("--- a/{}\n", path));
                diff_output.push_str("+++ /dev/null\n");

                let before_content = get_file_content(&handle.repo, &diff_values.before);
                for line in before_content.lines() {
                    diff_output.push_str(&format!("-{}\n", line));
                }
            }
            diff_output.push('\n');
        }
    });

    JjResult::success(diff_output)
}

fn get_file_content(
    repo: &Arc<ReadonlyRepo>,
    tree_value: &jj_lib::merge::Merge<Option<jj_lib::backend::TreeValue>>,
) -> String {
    use jj_lib::repo::Repo;
    use tokio::io::AsyncReadExt;

    // Try to get the first resolved value
    if let Some(Some(value)) = tree_value.as_resolved() {
        if let jj_lib::backend::TreeValue::File { id, .. } = value {
            let read_result = pollster::block_on(
                repo.store().read_file(&jj_lib::repo_path::RepoPath::root(), id),
            );
            if let Ok(mut reader) = read_result {
                let mut content = Vec::new();
                if pollster::block_on(reader.read_to_end(&mut content)).is_ok() {
                    return String::from_utf8_lossy(&content).to_string();
                }
            }
        }
    }
    String::new()
}

fn generate_unified_diff(before: &str, after: &str) -> String {
    let before_lines: Vec<&str> = before.lines().collect();
    let after_lines: Vec<&str> = after.lines().collect();

    let mut result = String::new();

    // Simple line-by-line diff (could be improved with proper diff algorithm)
    let max_lines = before_lines.len().max(after_lines.len());

    if max_lines == 0 {
        return result;
    }

    // Add a simple hunk header
    result.push_str(&format!(
        "@@ -1,{} +1,{} @@\n",
        before_lines.len(),
        after_lines.len()
    ));

    // Use a basic LCS-style diff
    let mut i = 0;
    let mut j = 0;

    while i < before_lines.len() || j < after_lines.len() {
        if i < before_lines.len() && j < after_lines.len() && before_lines[i] == after_lines[j] {
            result.push_str(&format!(" {}\n", before_lines[i]));
            i += 1;
            j += 1;
        } else if j < after_lines.len()
            && (i >= before_lines.len()
                || !before_lines[i..].contains(&after_lines[j]))
        {
            result.push_str(&format!("+{}\n", after_lines[j]));
            j += 1;
        } else if i < before_lines.len() {
            result.push_str(&format!("-{}\n", before_lines[i]));
            i += 1;
        }
    }

    result
}

/// Get diff for a specific file in the working copy
/// Returns JjResult with unified diff string on success
#[no_mangle]
pub extern "C" fn jj_get_file_diff(handle: *mut RepoHandle, path: *const c_char) -> JjResult {
    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &*handle
    };

    let path_str = unsafe {
        if path.is_null() {
            return JjResult::error("null path".to_string());
        }
        match CStr::from_ptr(path).to_str() {
            Ok(s) => s,
            Err(e) => return JjResult::error(format!("invalid UTF-8: {}", e)),
        }
    };

    // Find the current workspace's working copy commit
    let wc_commit_id = match handle
        .repo
        .view()
        .wc_commit_ids()
        .iter()
        .find(|(ws_id, _)| ws_id.as_str() == handle.current_workspace)
    {
        Some((_, commit_id)) => commit_id.clone(),
        None => return JjResult::error("No working copy found for current workspace".to_string()),
    };

    // Get the working copy commit
    let wc_commit: Commit = match handle.repo.store().get_commit(&wc_commit_id) {
        Ok(commit) => commit,
        Err(e) => return JjResult::error(format!("Failed to get working copy commit: {}", e)),
    };

    // Get the parent commit(s)
    let parent_ids = wc_commit.parent_ids();
    if parent_ids.is_empty() {
        return JjResult::success("".to_string());
    }

    let parent_commit: Commit = match handle.repo.store().get_commit(&parent_ids[0]) {
        Ok(commit) => commit,
        Err(e) => return JjResult::error(format!("Failed to get parent commit: {}", e)),
    };

    // Get trees for comparison
    let parent_tree: MergedTree = parent_commit.tree();
    let wc_tree: MergedTree = wc_commit.tree();

    // Build a matcher for just this file
    let repo_path = match jj_lib::repo_path::RepoPathBuf::from_internal_string(path_str) {
        Ok(p) => p,
        Err(e) => return JjResult::error(format!("Invalid path: {:?}", e)),
    };
    let matcher = jj_lib::matchers::FilesMatcher::new(vec![repo_path]);

    // Collect diff output
    let mut diff_output = String::new();

    let diff_stream = parent_tree.diff_stream(&wc_tree, &matcher);

    pollster::block_on(async {
        use futures_util::StreamExt;
        futures_util::pin_mut!(diff_stream);

        while let Some(entry) = diff_stream.next().await {
            let diff_values = match entry.values {
                Ok(v) => v,
                Err(_) => continue,
            };

            let entry_path = entry.path.as_internal_file_string();

            // Determine the change type
            let before_is_file = !diff_values.before.is_absent();
            let after_is_file = !diff_values.after.is_absent();

            if before_is_file && after_is_file {
                diff_output.push_str(&format!("diff --git a/{} b/{}\n", entry_path, entry_path));
                diff_output.push_str(&format!("--- a/{}\n", entry_path));
                diff_output.push_str(&format!("+++ b/{}\n", entry_path));

                let before_content = get_file_content(&handle.repo, &diff_values.before);
                let after_content = get_file_content(&handle.repo, &diff_values.after);

                let diff_lines = generate_unified_diff(&before_content, &after_content);
                diff_output.push_str(&diff_lines);
            } else if !before_is_file && after_is_file {
                diff_output.push_str(&format!("diff --git a/{} b/{}\n", entry_path, entry_path));
                diff_output.push_str("new file\n");
                diff_output.push_str("--- /dev/null\n");
                diff_output.push_str(&format!("+++ b/{}\n", entry_path));

                let after_content = get_file_content(&handle.repo, &diff_values.after);
                for line in after_content.lines() {
                    diff_output.push_str(&format!("+{}\n", line));
                }
            } else if before_is_file && !after_is_file {
                diff_output.push_str(&format!("diff --git a/{} b/{}\n", entry_path, entry_path));
                diff_output.push_str("deleted file\n");
                diff_output.push_str(&format!("--- a/{}\n", entry_path));
                diff_output.push_str("+++ /dev/null\n");

                let before_content = get_file_content(&handle.repo, &diff_values.before);
                for line in before_content.lines() {
                    diff_output.push_str(&format!("-{}\n", line));
                }
            }
        }
    });

    JjResult::success(diff_output)
}

/// Get the before/after content for a file in the working copy
/// Returns JjResult with JSON containing before and after content
#[no_mangle]
pub extern "C" fn jj_get_file_contents(
    handle: *mut RepoHandle,
    path: *const c_char,
) -> JjResult {
    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &*handle
    };

    let path_str = unsafe {
        if path.is_null() {
            return JjResult::error("null path".to_string());
        }
        match CStr::from_ptr(path).to_str() {
            Ok(s) => s,
            Err(e) => return JjResult::error(format!("invalid UTF-8: {}", e)),
        }
    };

    // Find the current workspace's working copy commit
    let wc_commit_id = match handle
        .repo
        .view()
        .wc_commit_ids()
        .iter()
        .find(|(ws_id, _)| ws_id.as_str() == handle.current_workspace)
    {
        Some((_, commit_id)) => commit_id.clone(),
        None => return JjResult::error("No working copy found for current workspace".to_string()),
    };

    // Get the working copy commit
    let wc_commit: Commit = match handle.repo.store().get_commit(&wc_commit_id) {
        Ok(commit) => commit,
        Err(e) => return JjResult::error(format!("Failed to get working copy commit: {}", e)),
    };

    // Get the parent commit(s)
    let parent_ids = wc_commit.parent_ids();
    if parent_ids.is_empty() {
        // No parent, return empty before content
        let contents = FileContents {
            before: String::new(),
            after: String::new(),
            path: path_str.to_string(),
        };
        return match serde_json::to_string(&contents) {
            Ok(json) => JjResult::success(json),
            Err(e) => JjResult::error(format!("JSON serialization failed: {}", e)),
        };
    }

    let parent_commit: Commit = match handle.repo.store().get_commit(&parent_ids[0]) {
        Ok(commit) => commit,
        Err(e) => return JjResult::error(format!("Failed to get parent commit: {}", e)),
    };

    // Build a repo path
    let repo_path = match jj_lib::repo_path::RepoPathBuf::from_internal_string(path_str) {
        Ok(p) => p,
        Err(e) => return JjResult::error(format!("Invalid path: {:?}", e)),
    };

    // Get trees for comparison
    let parent_tree: MergedTree = parent_commit.tree();
    let wc_tree: MergedTree = wc_commit.tree();

    // Get the file content at this path from both trees
    let before_value = match parent_tree.path_value(&repo_path) {
        Ok(v) => v,
        Err(e) => return JjResult::error(format!("Failed to get before value: {}", e)),
    };
    let after_value = match wc_tree.path_value(&repo_path) {
        Ok(v) => v,
        Err(e) => return JjResult::error(format!("Failed to get after value: {}", e)),
    };

    let before_content = get_file_content(&handle.repo, &before_value);
    let after_content = get_file_content(&handle.repo, &after_value);

    let contents = FileContents {
        before: before_content,
        after: after_content,
        path: path_str.to_string(),
    };

    match serde_json::to_string(&contents) {
        Ok(json) => JjResult::success(json),
        Err(e) => JjResult::error(format!("JSON serialization failed: {}", e)),
    }
}

/// Get diff for a revision compared to its parent
/// Returns JjResult with unified diff string on success
#[no_mangle]
pub extern "C" fn jj_get_revision_diff(handle: *mut RepoHandle, revision_id: *const c_char) -> JjResult {
    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &*handle
    };

    let revision_str = unsafe {
        if revision_id.is_null() {
            return JjResult::error("null revision_id".to_string());
        }
        match CStr::from_ptr(revision_id).to_str() {
            Ok(s) => s,
            Err(e) => return JjResult::error(format!("invalid UTF-8: {}", e)),
        }
    };

    // Find the commit by ID prefix - walk from working copy commits
    let commit = {
        use std::collections::HashSet;
        let mut found: Option<Commit> = None;
        let mut visited: HashSet<String> = HashSet::new();
        let mut to_visit: Vec<jj_lib::backend::CommitId> = Vec::new();

        // Start from all working copy commits
        for (_ws_id, commit_id) in handle.repo.view().wc_commit_ids() {
            to_visit.push(commit_id.clone());
        }

        while let Some(commit_id) = to_visit.pop() {
            let hex = commit_id.hex();
            if visited.contains(&hex) {
                continue;
            }
            visited.insert(hex.clone());

            if hex.starts_with(revision_str) {
                match handle.repo.store().get_commit(&commit_id) {
                    Ok(c) => {
                        found = Some(c);
                        break;
                    }
                    Err(_) => continue,
                }
            }

            // Add parents to visit
            if let Ok(c) = handle.repo.store().get_commit(&commit_id) {
                for parent_id in c.parent_ids() {
                    if !visited.contains(&parent_id.hex()) {
                        to_visit.push(parent_id.clone());
                    }
                }
            }
        }

        match found {
            Some(c) => c,
            None => return JjResult::error(format!("Revision not found: {}", revision_str)),
        }
    };

    // Get the parent commit(s)
    let parent_ids = commit.parent_ids();
    if parent_ids.is_empty() {
        return JjResult::success("".to_string());
    }

    let parent_commit: Commit = match handle.repo.store().get_commit(&parent_ids[0]) {
        Ok(c) => c,
        Err(e) => return JjResult::error(format!("Failed to get parent commit: {}", e)),
    };

    // Get trees for comparison
    let parent_tree: MergedTree = parent_commit.tree();
    let commit_tree: MergedTree = commit.tree();

    // Collect diff output
    let mut diff_output = String::new();
    let matcher = EverythingMatcher;

    let diff_stream = parent_tree.diff_stream(&commit_tree, &matcher);

    pollster::block_on(async {
        use futures_util::StreamExt;
        futures_util::pin_mut!(diff_stream);

        while let Some(entry) = diff_stream.next().await {
            let diff_values = match entry.values {
                Ok(v) => v,
                Err(_) => continue,
            };

            let path = entry.path.as_internal_file_string();

            let before_is_file = !diff_values.before.is_absent();
            let after_is_file = !diff_values.after.is_absent();

            if before_is_file && after_is_file {
                diff_output.push_str(&format!("diff --git a/{} b/{}\n", path, path));
                diff_output.push_str(&format!("--- a/{}\n", path));
                diff_output.push_str(&format!("+++ b/{}\n", path));

                let before_content = get_file_content(&handle.repo, &diff_values.before);
                let after_content = get_file_content(&handle.repo, &diff_values.after);

                let diff_lines = generate_unified_diff(&before_content, &after_content);
                diff_output.push_str(&diff_lines);
            } else if !before_is_file && after_is_file {
                diff_output.push_str(&format!("diff --git a/{} b/{}\n", path, path));
                diff_output.push_str("new file\n");
                diff_output.push_str("--- /dev/null\n");
                diff_output.push_str(&format!("+++ b/{}\n", path));

                let after_content = get_file_content(&handle.repo, &diff_values.after);
                for line in after_content.lines() {
                    diff_output.push_str(&format!("+{}\n", line));
                }
            } else if before_is_file && !after_is_file {
                diff_output.push_str(&format!("diff --git a/{} b/{}\n", path, path));
                diff_output.push_str("deleted file\n");
                diff_output.push_str(&format!("--- a/{}\n", path));
                diff_output.push_str("+++ /dev/null\n");

                let before_content = get_file_content(&handle.repo, &diff_values.before);
                for line in before_content.lines() {
                    diff_output.push_str(&format!("-{}\n", line));
                }
            }
            diff_output.push('\n');
        }
    });

    JjResult::success(diff_output)
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
