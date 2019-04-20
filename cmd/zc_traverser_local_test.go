package cmd

import (
	chk "gopkg.in/check.v1"
)

type localTraverserTestSuite struct{}

var _ = chk.Suite(&localTraverserTestSuite{})

func (s *localTraverserTestSuite) TestCleanLocalPath(c *chk.C) {
	testCases := map[string]string{
		"C:\\foo\\bar":   "C:/foo/bar",    // regular windows path with no change
		"C:\\foo\\bar\\": "C:/foo/bar",    // regular windows path with extra slash
		".\\foo\\bar":    "foo/bar",       // relative windows path
		"..\\foo\\bar":   "../foo/bar",    // relative windows path with parent dir
		"foo\\bar":       "foo/bar",       // shorthand relative windows path
		"\\\\foo\\bar\\": "//foo/bar",     // network share
		"C:\\":           "C:/",           // special case, the slash after colon is actually required
		"D:":             "D:/",           // special case, the slash after colon is actually required
		"/user/foo/bar":  "/user/foo/bar", // regular unix path with no change
		"/user/foo/bar/": "/user/foo/bar", // regular unix path with extra slash
		"./foo/bar":      "foo/bar",       // relative unix path
		"../foo/bar":     "../foo/bar",    // relative unix path with parent dir
		"foo/bar":        "foo/bar",       // shorthand relative unix path
	}

	for orig, expected := range testCases {
		c.Assert(cleanLocalPath(orig), chk.Equals, expected)
	}
}
