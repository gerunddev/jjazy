package fixtures

// FileStatus represents the status of a file change
type FileStatus int

const (
	StatusModified FileStatus = iota
	StatusAdded
	StatusDeleted
	StatusRenamed
	StatusConflict
)

func (s FileStatus) String() string {
	switch s {
	case StatusModified:
		return "M"
	case StatusAdded:
		return "A"
	case StatusDeleted:
		return "D"
	case StatusRenamed:
		return "R"
	case StatusConflict:
		return "C"
	default:
		return "?"
	}
}

// Workspace represents a jj workspace
type Workspace struct {
	Name        string
	IsCurrent   bool
	RevisionID  string
	ChangeID    string
	Description string
}

// FileChange represents a changed file
type FileChange struct {
	Path   string
	Status FileStatus
}

// Bookmark represents a jj bookmark (branch)
type Bookmark struct {
	Name      string
	IsLocal   bool
	RevisionID string
	IsCurrent bool
}

// Operation represents a jj operation in the undo history
type Operation struct {
	ID          string
	Description string
	Timestamp   string
	IsCurrent   bool
}

// Revision represents a commit/revision in the log
type Revision struct {
	ID            string
	ChangeID      string
	Description   string
	Author        string
	Timestamp     string
	Bookmarks     []string
	GitHead       bool
	IsWorkingCopy bool
	WorkspaceName string
	IsRoot        bool
	Parents       []string
}

// Mock data for development

var Workspaces = []Workspace{
	{
		Name:        "default",
		IsCurrent:   true,
		RevisionID:  "qvtrunyp",
		ChangeID:    "kxryzmor",
		Description: "Add user authentication flow",
	},
	{
		Name:        "feature-auth",
		IsCurrent:   false,
		RevisionID:  "mxklpnrs",
		ChangeID:    "ywtqplmn",
		Description: "Refactor auth middleware",
	},
	{
		Name:        "experiment",
		IsCurrent:   false,
		RevisionID:  "zzpqwert",
		ChangeID:    "abcdwxyz",
		Description: "Try new caching approach",
	},
}

var Files = []FileChange{
	{Path: "src/auth/login.rs", Status: StatusModified},
	{Path: "src/auth/session.rs", Status: StatusModified},
	{Path: "src/auth/oauth.rs", Status: StatusAdded},
	{Path: "src/deprecated.rs", Status: StatusDeleted},
	{Path: "src/utils/helpers.rs", Status: StatusRenamed},
	{Path: "src/config.rs", Status: StatusConflict},
}

var Bookmarks = []Bookmark{
	{Name: "main", IsLocal: true, RevisionID: "abc12345", IsCurrent: false},
	{Name: "feature-login", IsLocal: true, RevisionID: "qvtrunyp", IsCurrent: true},
	{Name: "feature-oauth", IsLocal: true, RevisionID: "mxklpnrs", IsCurrent: false},
	{Name: "origin/main", IsLocal: false, RevisionID: "abc12345", IsCurrent: false},
	{Name: "origin/develop", IsLocal: false, RevisionID: "def67890", IsCurrent: false},
}

var Operations = []Operation{
	{ID: "op1", Description: "new commit", Timestamp: "2 minutes ago", IsCurrent: true},
	{ID: "op2", Description: "describe", Timestamp: "5 minutes ago", IsCurrent: false},
	{ID: "op3", Description: "edit commit", Timestamp: "10 minutes ago", IsCurrent: false},
	{ID: "op4", Description: "squash", Timestamp: "15 minutes ago", IsCurrent: false},
	{ID: "op5", Description: "rebase", Timestamp: "1 hour ago", IsCurrent: false},
}

var Log = []Revision{
	{
		ID:            "qvtrunyp",
		ChangeID:      "kxryzmor",
		Description:   "Add user authentication flow",
		Author:        "ben",
		Timestamp:     "2 minutes ago",
		IsWorkingCopy: true,
		WorkspaceName: "default",
		Parents:       []string{"pqwertyu"},
	},
	{
		ID:            "mxklpnrs",
		ChangeID:      "ywtqplmn",
		Description:   "Refactor auth middleware",
		Author:        "ben",
		Timestamp:     "1 hour ago",
		IsWorkingCopy: true,
		WorkspaceName: "feature-auth",
		Parents:       []string{"pqwertyu"},
	},
	{
		ID:          "pqwertyu",
		ChangeID:    "mnbvcxzl",
		Description: "Fix bug in parser",
		Author:      "ben",
		Timestamp:   "2 hours ago",
		Parents:     []string{"abc12345"},
	},
	{
		ID:          "abc12345",
		ChangeID:    "zxcvbnma",
		Description: "Initial commit",
		Author:      "ben",
		Timestamp:   "1 day ago",
		IsRoot:      true,
		Parents:     []string{},
	},
}

var DiffContent = `diff --git a/src/auth/login.rs b/src/auth/login.rs
index 1234567..abcdefg 100644
--- a/src/auth/login.rs
+++ b/src/auth/login.rs
@@ -1,10 +1,15 @@
 use crate::auth::session::Session;
+use crate::auth::oauth::OAuthProvider;

 pub struct LoginHandler {
     session: Session,
+    oauth: Option<OAuthProvider>,
 }

 impl LoginHandler {
-    pub fn new() -> Self {
-        Self { session: Session::new() }
+    pub fn new(oauth: Option<OAuthProvider>) -> Self {
+        Self {
+            session: Session::new(),
+            oauth,
+        }
     }

@@ -15,6 +20,14 @@ impl LoginHandler {
         self.session.validate(token)
     }

+    pub fn oauth_login(&self, provider: &str) -> Result<String, AuthError> {
+        match &self.oauth {
+            Some(oauth) => oauth.authenticate(provider),
+            None => Err(AuthError::OAuthNotConfigured),
+        }
+    }
+
     pub fn logout(&mut self) {
         self.session.invalidate();
     }
`
