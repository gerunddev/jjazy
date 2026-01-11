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
    repo_root: String, // Directory where repo was opened
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
    root_path: String, // Absolute path to workspace directory
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

    // Start with jj-lib's built-in defaults (includes experimental settings)
    let mut config = StackedConfig::with_defaults();

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

        [experimental]
        record-predecessors-in-commit = false
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
            let workspace_root = workspace.workspace_root().to_string_lossy().to_string();
            match workspace.repo_loader().load_at_head() {
                Ok(repo) => {
                    let handle = Box::new(RepoHandle {
                        repo,
                        current_workspace: workspace_name,
                        repo_root: workspace_root,
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

    // Get the parent directory of current workspace for computing sibling paths
    let repo_root_path = Path::new(&handle.repo_root);
    let parent_dir = repo_root_path.parent();

    // Get all workspaces from the view's working copy commit IDs
    for (workspace_id, commit_id) in handle.repo.view().wc_commit_ids() {
        let ws_name = workspace_id.as_str().to_string();
        let is_current = ws_name == handle.current_workspace;

        // Compute root_path:
        // - For current workspace: use repo_root
        // - For other workspaces: use sibling directory convention (parent_dir/workspace_name)
        let root_path = if is_current {
            handle.repo_root.clone()
        } else if let Some(parent) = parent_dir {
            parent.join(&ws_name).to_string_lossy().to_string()
        } else {
            // Fallback: just use workspace name (relative path)
            ws_name.clone()
        };

        workspaces.push(WorkspaceInfo {
            name: ws_name,
            is_current,
            commit_id: commit_id.hex(),
            root_path,
        });
    }

    // Sort workspaces by name for consistent ordering
    workspaces.sort_by(|a, b| a.name.cmp(&b.name));

    match serde_json::to_string(&workspaces) {
        Ok(json) => JjResult::success(json),
        Err(e) => JjResult::error(format!("JSON serialization failed: {}", e)),
    }
}

/// Get the parent commit ID(s) of the current workspace's working copy
fn get_current_wc_parent_ids(handle: &RepoHandle) -> Result<Vec<jj_lib::backend::CommitId>, String> {
    // Find current workspace's working copy commit
    let wc_commit_id = handle.repo.view().wc_commit_ids()
        .iter()
        .find(|(ws_id, _)| ws_id.as_str() == handle.current_workspace)
        .map(|(_, commit_id)| commit_id.clone())
        .ok_or_else(|| "No working copy found for current workspace".to_string())?;

    // Get the commit
    let wc_commit = handle.repo.store().get_commit(&wc_commit_id)
        .map_err(|e| format!("Failed to get working copy commit: {}", e))?;

    // Return parent IDs
    Ok(wc_commit.parent_ids().to_vec())
}

/// Resolve revision spec strings (commit ID prefixes) to commit IDs.
/// The search is limited to MAX_REVISION_SEARCH_DEPTH commits to avoid
/// unbounded walks in very large repositories.
const MAX_REVISION_SEARCH_DEPTH: usize = 10000;

fn resolve_revision_specs(handle: &RepoHandle, specs: &[String]) -> Result<Vec<jj_lib::backend::CommitId>, String> {
    use std::collections::HashSet;

    let mut result = Vec::new();

    for spec in specs {
        // Walk from working copy commits to find matching revision
        let mut found: Option<jj_lib::backend::CommitId> = None;
        let mut visited: HashSet<String> = HashSet::new();
        let mut to_visit: Vec<jj_lib::backend::CommitId> = Vec::new();

        for (_ws_id, commit_id) in handle.repo.view().wc_commit_ids() {
            to_visit.push(commit_id.clone());
        }

        while let Some(commit_id) = to_visit.pop() {
            // Depth limit to avoid unbounded walks in large repos
            if visited.len() >= MAX_REVISION_SEARCH_DEPTH {
                break;
            }

            let hex = commit_id.hex();
            if visited.contains(&hex) {
                continue;
            }
            visited.insert(hex.clone());

            if hex.starts_with(spec) {
                found = Some(commit_id);
                break;
            }

            if let Ok(c) = handle.repo.store().get_commit(&commit_id) {
                for parent_id in c.parent_ids() {
                    if !visited.contains(&parent_id.hex()) {
                        to_visit.push(parent_id.clone());
                    }
                }
            }
        }

        match found {
            Some(id) => result.push(id),
            None => return Err(format!("Revision not found: {}", spec)),
        }
    }

    Ok(result)
}

/// Add a new workspace at the given path
/// Creates directory if needed, initializes workspace with existing repo
/// If revision_ids is NULL or empty, the new workspace will be created as a sibling
/// of the current workspace's working copy (sharing the same parent commits).
/// If revision_ids is provided (comma-separated), those commits become the parents.
/// Returns JjResult with empty success or error message
#[no_mangle]
pub extern "C" fn jj_workspace_add(
    handle: *mut RepoHandle,
    destination_path: *const c_char,
    workspace_name: *const c_char,
    revision_ids: *const c_char,
) -> JjResult {
    use jj_lib::ref_name::WorkspaceNameBuf;
    use jj_lib::local_working_copy::LocalWorkingCopyFactory;
    use std::fs;

    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &mut *handle
    };

    let dest_path_str = unsafe {
        if destination_path.is_null() {
            return JjResult::error("null destination_path".to_string());
        }
        match CStr::from_ptr(destination_path).to_str() {
            Ok(s) => s,
            Err(e) => return JjResult::error(format!("invalid destination_path UTF-8: {}", e)),
        }
    };

    // Parse revision IDs (comma-separated or NULL for default)
    let revision_specs: Vec<String> = unsafe {
        if revision_ids.is_null() {
            Vec::new()  // Empty means "use default" (parent of current @)
        } else {
            match CStr::from_ptr(revision_ids).to_str() {
                Ok(s) if s.is_empty() => Vec::new(),
                Ok(s) => s.split(',').map(|r| r.trim().to_string()).filter(|s| !s.is_empty()).collect(),
                Err(e) => return JjResult::error(format!("invalid revision_ids UTF-8: {}", e)),
            }
        }
    };

    // Get workspace name - use provided name or derive from path basename
    let ws_name = unsafe {
        if workspace_name.is_null() {
            // Derive from destination path basename
            Path::new(dest_path_str)
                .file_name()
                .and_then(|n| n.to_str())
                .unwrap_or("default")
                .to_string()
        } else {
            match CStr::from_ptr(workspace_name).to_str() {
                Ok(s) => s.to_string(),
                Err(e) => return JjResult::error(format!("invalid workspace_name UTF-8: {}", e)),
            }
        }
    };

    // Convert to absolute path
    let dest_path = Path::new(dest_path_str);
    let abs_dest_path = if dest_path.is_absolute() {
        dest_path.to_path_buf()
    } else {
        // Resolve relative to repo root's parent
        Path::new(&handle.repo_root)
            .parent()
            .unwrap_or(Path::new("/"))
            .join(dest_path)
    };

    // Validate path: must not exist OR be an empty directory
    if abs_dest_path.exists() {
        if abs_dest_path.is_file() {
            return JjResult::error(format!("Path exists and is a file: {}", abs_dest_path.display()));
        }
        // Check if directory is empty or only contains .jj
        match fs::read_dir(&abs_dest_path) {
            Ok(entries) => {
                let non_jj_entries: Vec<_> = entries
                    .filter_map(|e| e.ok())
                    .filter(|e| e.file_name() != ".jj")
                    .collect();
                if !non_jj_entries.is_empty() {
                    return JjResult::error(format!(
                        "Directory is not empty: {}",
                        abs_dest_path.display()
                    ));
                }
            }
            Err(e) => {
                return JjResult::error(format!(
                    "Cannot read directory {}: {}",
                    abs_dest_path.display(),
                    e
                ));
            }
        }
    } else {
        // Create the directory
        if let Err(e) = fs::create_dir_all(&abs_dest_path) {
            return JjResult::error(format!(
                "Failed to create directory {}: {}",
                abs_dest_path.display(),
                e
            ));
        }
    }

    // Determine parent commit(s) for new workspace's working copy
    let parent_ids: Vec<jj_lib::backend::CommitId> = if revision_specs.is_empty() {
        // Default: use parent(s) of current workspace's working copy
        match get_current_wc_parent_ids(handle) {
            Ok(ids) => ids,
            Err(e) => return JjResult::error(e),
        }
    } else {
        // Explicit: resolve the specified revision(s)
        match resolve_revision_specs(handle, &revision_specs) {
            Ok(ids) => ids,
            Err(e) => return JjResult::error(e),
        }
    };

    // Handle edge case: if no parents (root commit scenario), use root
    let parent_ids = if parent_ids.is_empty() {
        vec![handle.repo.store().root_commit_id().clone()]
    } else {
        parent_ids
    };

    // Initialize workspace with existing repo
    let workspace_name_buf = WorkspaceNameBuf::from(ws_name.clone());
    let working_copy_factory = LocalWorkingCopyFactory {};

    // The repo path is at workspace_root/.jj/repo
    let repo_path = Path::new(&handle.repo_root).join(".jj").join("repo");

    match Workspace::init_workspace_with_existing_repo(
        &abs_dest_path,
        &repo_path,
        &handle.repo,
        &working_copy_factory,
        workspace_name_buf.clone(),
    ) {
        Ok((workspace, new_repo)) => {
            // Now we need to update the new workspace's working copy to have correct parents
            // Start a transaction to create the new working copy commit
            let mut tx = new_repo.start_transaction();

            // Get parent commits
            let parents: Vec<Commit> = parent_ids.iter()
                .filter_map(|id| tx.repo().store().get_commit(id).ok())
                .collect();

            if parents.is_empty() {
                return JjResult::error("Failed to resolve parent commits".to_string());
            }

            // Get the tree from first parent
            let parent_tree: MergedTree = parents[0].tree();

            // Create new empty commit with correct parents
            let new_commit = tx.repo_mut().new_commit(
                parents.iter().map(|c| c.id().clone()).collect(),
                parent_tree,
            ).write();

            let new_commit = match new_commit {
                Ok(c) => c,
                Err(e) => return JjResult::error(format!("Failed to write commit: {}", e)),
            };

            // Set this as the workspace's working copy
            if let Err(e) = tx.repo_mut().set_wc_commit(workspace_name_buf.clone(), new_commit.id().clone()) {
                return JjResult::error(format!("Failed to set working copy: {:?}", e));
            }

            // Commit the transaction
            let parent_hexes: Vec<String> = parent_ids.iter()
                .map(|id| {
                    let hex = id.hex();
                    hex[..8.min(hex.len())].to_string()
                })
                .collect();
            let description = format!("create workspace {} at {}", ws_name, parent_hexes.join(", "));
            match tx.commit(&description) {
                Ok(final_repo) => {
                    handle.repo = final_repo;
                    let _ = workspace; // Workspace is consumed
                    JjResult::success("".to_string())
                }
                Err(e) => JjResult::error(format!("Failed to commit transaction: {}", e)),
            }
        }
        Err(e) => JjResult::error(format!("Failed to create workspace: {:?}", e)),
    }
}

/// Forget a workspace by name
/// Removes workspace tracking from repo (does not delete files on disk)
/// Returns JjResult with empty success or error message
#[no_mangle]
pub extern "C" fn jj_workspace_forget(
    handle: *mut RepoHandle,
    workspace_name: *const c_char,
) -> JjResult {
    use jj_lib::ref_name::WorkspaceNameBuf;

    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &mut *handle
    };

    let ws_name = unsafe {
        if workspace_name.is_null() {
            return JjResult::error("null workspace_name".to_string());
        }
        match CStr::from_ptr(workspace_name).to_str() {
            Ok(s) => s,
            Err(e) => return JjResult::error(format!("invalid workspace_name UTF-8: {}", e)),
        }
    };

    // Cannot forget current workspace
    if ws_name == handle.current_workspace {
        return JjResult::error("Cannot forget current workspace".to_string());
    }

    // Check if workspace exists by iterating through wc_commit_ids
    let workspace_exists = handle.repo.view().wc_commit_ids()
        .iter()
        .any(|(ws_id, _)| ws_id.as_str() == ws_name);

    if !workspace_exists {
        return JjResult::error(format!("Workspace not found: {}", ws_name));
    }

    // Create workspace ID using WorkspaceNameBuf
    let workspace_name_buf = WorkspaceNameBuf::from(ws_name.to_string());

    // Start a transaction to remove the workspace
    let mut tx = handle.repo.start_transaction();

    // Remove the working copy commit for this workspace
    if let Err(e) = tx.repo_mut().remove_wc_commit(&workspace_name_buf) {
        return JjResult::error(format!("Failed to remove workspace: {:?}", e));
    }

    // Commit the transaction
    let description = format!("forget workspace {}", ws_name);
    match tx.commit(&description) {
        Ok(new_repo) => {
            handle.repo = new_repo;
            JjResult::success("".to_string())
        }
        Err(e) => JjResult::error(format!("Failed to commit transaction: {}", e)),
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

/// Set a bookmark to point to a specific revision.
///
/// Parameters:
/// - handle: repository handle
/// - name: bookmark name (C string)
/// - revision_id: target revision ID prefix (C string)
/// - allow_backwards: if true, allow moving bookmark backwards in history
/// - ignore_immutable: if true, allow setting bookmark on immutable revisions
///
/// Returns JjResult with empty success or error message.
#[no_mangle]
pub extern "C" fn jj_set_bookmark(
    handle: *mut RepoHandle,
    name: *const c_char,
    revision_id: *const c_char,
    allow_backwards: bool,
    ignore_immutable: bool,
) -> JjResult {
    use std::collections::HashSet;
    use jj_lib::op_store::RefTarget;
    use jj_lib::ref_name::RefNameBuf;

    let handle = unsafe {
        if handle.is_null() {
            return JjResult::error("null repo handle".to_string());
        }
        &mut *handle
    };

    let bookmark_name = unsafe {
        if name.is_null() {
            return JjResult::error("null bookmark name".to_string());
        }
        match CStr::from_ptr(name).to_str() {
            Ok(s) => s,
            Err(e) => return JjResult::error(format!("invalid bookmark name UTF-8: {}", e)),
        }
    };

    // Convert bookmark name to RefNameBuf
    let ref_name = RefNameBuf::from(bookmark_name.to_string());

    let revision_str = unsafe {
        if revision_id.is_null() {
            return JjResult::error("null revision_id".to_string());
        }
        match CStr::from_ptr(revision_id).to_str() {
            Ok(s) => s,
            Err(e) => return JjResult::error(format!("invalid revision_id UTF-8: {}", e)),
        }
    };

    // Find the target commit by ID prefix
    let target_commit = {
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

    // Check if target is immutable
    // Immutable commits are those that are ancestors of:
    // 1. Remote-tracked bookmarks (pushed commits)
    // 2. The root commit
    if !ignore_immutable {
        let root_commit_id = handle.repo.store().root_commit_id();
        if target_commit.id() == root_commit_id {
            return JjResult::error("Cannot set bookmark on immutable revision (root commit)".to_string());
        }

        // Check if commit is an ancestor of any remote-tracked bookmark
        // This covers the common case: commits that have been pushed are immutable
        let mut remote_heads: Vec<jj_lib::backend::CommitId> = Vec::new();

        // Collect all remote bookmark targets
        for (_, remote_ref) in handle.repo.view().all_remote_bookmarks() {
            for id in remote_ref.target.added_ids() {
                remote_heads.push(id.clone());
            }
        }

        // Check if target_commit is an ancestor of any remote head
        // by walking down from remote heads to see if we reach target
        if !remote_heads.is_empty() {
            let mut is_immutable = false;
            let mut to_check: Vec<jj_lib::backend::CommitId> = remote_heads;
            let mut checked: HashSet<String> = HashSet::new();
            let max_depth = 200;

            for _ in 0..max_depth {
                if to_check.is_empty() {
                    break;
                }

                let commit_id = to_check.pop().unwrap();
                let hex = commit_id.hex();
                if checked.contains(&hex) {
                    continue;
                }
                checked.insert(hex);

                if &commit_id == target_commit.id() {
                    // Target is an ancestor of remote bookmark - it's immutable
                    is_immutable = true;
                    break;
                }

                // Add parents to check
                if let Ok(c) = handle.repo.store().get_commit(&commit_id) {
                    for parent_id in c.parent_ids() {
                        to_check.push(parent_id.clone());
                    }
                }
            }

            if is_immutable {
                return JjResult::error("Cannot set bookmark on immutable revision (already pushed)".to_string());
            }
        }
    }

    // Check if moving backwards
    if !allow_backwards {
        // Get current bookmark target
        let current_target = handle.repo.view().get_local_bookmark(&ref_name);
        if let Some(ref_target) = current_target.as_normal() {
            // Check if new target is an ancestor of current target
            if let Ok(current_commit) = handle.repo.store().get_commit(ref_target) {
                // Simple ancestor check: walk from current to see if we reach target
                let mut is_backwards = false;
                let mut ancestors_to_check: Vec<jj_lib::backend::CommitId> = vec![current_commit.id().clone()];
                let mut checked: HashSet<String> = HashSet::new();
                let max_depth = 100; // Limit search depth

                for _ in 0..max_depth {
                    if ancestors_to_check.is_empty() {
                        break;
                    }

                    let commit_id = ancestors_to_check.pop().unwrap();
                    let hex = commit_id.hex();
                    if checked.contains(&hex) {
                        continue;
                    }
                    checked.insert(hex);

                    if &commit_id == target_commit.id() {
                        // Target is an ancestor of current - this is backwards
                        is_backwards = true;
                        break;
                    }

                    if let Ok(c) = handle.repo.store().get_commit(&commit_id) {
                        for parent_id in c.parent_ids() {
                            ancestors_to_check.push(parent_id.clone());
                        }
                    }
                }

                if is_backwards {
                    return JjResult::error("Cannot move bookmark backwards (use allow_backwards flag)".to_string());
                }
            }
        }
    }

    // Start a transaction
    let mut tx = handle.repo.start_transaction();

    // Set the bookmark
    tx.repo_mut().set_local_bookmark_target(
        &ref_name,
        RefTarget::normal(target_commit.id().clone()),
    );

    // Commit the transaction
    let description = format!("set bookmark {} to {}", bookmark_name, revision_str);
    match tx.commit(&description) {
        Ok(new_repo) => {
            // Update handle to point to new repo
            handle.repo = new_repo;
            JjResult::success("".to_string())
        }
        Err(e) => JjResult::error(format!("Failed to commit transaction: {}", e)),
    }
}
