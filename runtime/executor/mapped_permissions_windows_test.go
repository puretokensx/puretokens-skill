package main

import (
	"golang.org/x/sys/windows"
	"path/filepath"
	"testing"
)

func TestMappedConnectionRejectsPublicWindowsACL(t *testing.T) {
	for _, broad := range []bool{false, true} {
		name := "private"
		if broad {
			name = "world-readable"
		}
		t.Run(name, func(t *testing.T) {
			path := clientFixtureFile(t, clientFixtureRoot(t), "connection.json", "{}")
			setWindowsMappedFixturePermissions(t, path, broad)
			if privateConnectionFile(path) == broad {
				t.Fatal("Windows connection permissions not enforced")
			}
		})
	}
}

func privateMappedFixtureFile(t *testing.T, path string) {
	t.Helper()
	setWindowsMappedFixturePermissions(t, path, false)
}

func setWindowsMappedFixturePermissions(t *testing.T, path string, broad bool) {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal("current user unavailable")
	}
	sddl := "D:P(A;;FA;;;" + user.User.Sid.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)"
	if broad {
		sddl += "(A;;FR;;;WD)"
	}
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		t.Fatal("invalid test descriptor")
	}
	acl, _, err := sd.DACL()
	if err != nil {
		t.Fatal("test DACL unavailable")
	}
	// Elevated runners can default to the Administrators owner; Unix mode bits
	// do not establish the current-user ownership and DACL this contract requires.
	const flags = windows.OWNER_SECURITY_INFORMATION | windows.DACL_SECURITY_INFORMATION | windows.PROTECTED_DACL_SECURITY_INFORMATION
	if windows.SetNamedSecurityInfo(filepath.Clean(path), windows.SE_FILE_OBJECT, flags, user.User.Sid, nil, acl, nil) != nil {
		t.Fatal("cannot set fixture owner and DACL")
	}
}
