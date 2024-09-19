package main

import "time"

const (
	sessionCookie = "oauthsession"
	stateCookie   = "oauthstate"

	sessionStateMaxAge  int = 120
	sessionCookieMaxAge int = 28800

	defaultSessionStoreSize         int = 8192
	defaultUserCheckIntervalSeconds int = 3600

	sessionGroupKey = "groups"
	sessionTokenKey = "token"
	sessionTSKey    = "ts"

	readerGroup = "seti-viewer-dev"

	shutdownTimeout             = 10 * time.Second
	defaultSessionMaxAgeSeconds = 14400
	defaultCleanupSeconds       = 3600
	oidcScopeSeti               = "seti"
)
