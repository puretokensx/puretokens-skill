package main

import (
	"golang.org/x/sys/windows"
	"path/filepath"
	"testing"
)

func TestMappedConnectionRejectsPublicWindowsACL(t *testing.T) {
	path := clientFixtureFile(t, clientFixtureRoot(t), "connection.json", "{}")
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal("current user unavailable")
	}
	for _, broad := range []bool{false, true} {
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
		if windows.SetNamedSecurityInfo(filepath.Clean(path), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil) != nil {
			t.Fatal("cannot set fixture DACL")
		}
		if privateConnectionFile(path) == broad {
			t.Fatal("Windows connection permissions not enforced")
		}
	}
}
