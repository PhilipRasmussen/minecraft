// Package versions fetches Mojang's versions listing of Minecraft, allowing
// clients to determine what the latest version is.
//
// This is useful to determine whether the latest version of the game is
// installed and/or construct download URLs for the latest versions of the game.
// For example:
//	vs, err := versions.Load(context.TODO())
//	if err != nil {
// 		log.Fatal("Failed to fetch versions listing: " + err.Error())
//	}
//
//	if latest := vs.Latest.Release; latest != currentVersion {
// 		url := fmt.Sprintf("http://s3.amazonaws.com/Minecraft.Download/versions/%s/%s.jar", latest, latest)
//		resp, err := http.Get(url)
//		...
//	}
// For more information, see: http://wiki.vg/Game_Files
package versions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Load fetches a listing of Minecraft versions from Mojang's servers.
// ctx must be non-nil. If an error occurs, l will be uninitialised.
// Load reports Mojang server communication failures using *url.Error.
func Load(ctx context.Context) (l Listing, err error) {
	m, err := fetchJSON(ctx)
	if err == nil {
		err = initialize(&l, m)
		if err != nil {
			l = Listing{}
		}
	}
	return
}

// Listing represents a listing of Minecraft versions.
type Listing struct {
	Versions map[string]Version // Every known version of Minecraft indexed by version ID
	Latest   struct {           // The IDs of the latest snapshot and release versions.
		Snapshot string // The version ID of the latest development snapshot
		Release  string // The version ID of the latest Minecraft release
	}
}

// LatestRelease returns the version information for the latest release version.
// It is the same as l.Versions[l.Latest.Release], except LatestRelease will
// panic if l.Versions doesn't contain the key l.Latest.Release.
func (l Listing) LatestRelease() Version {
	v, ok := l.Versions[l.Latest.Release]
	if ok {
		return v
	} else {
		panic("l.Versions does not contain the l.Latest.Release key " + l.Latest.Release)
	}
}

///////////////////

// Version contains information about a Minecraft version.
// Version values should be used as map or database keys with caution as they
// contain a time.Time field. Using ID as the key alone is recommended.
// For the same reasons, do not use == with Version values; use Equal instead.
type Version struct {
	ID       string    // E.g. "1.8.1"
	Released time.Time // Time the version was released
	Type     Type      // Type of version, e.g. release or snapshot
}

// Equal reports whether v and u represents the same Minecraft version.
// For this to be true, v and u must have the same ID, be released at
// the same time instant, and be of the same release type.
// Do not use == with Version values.
func (v Version) Equal(u Version) bool {
	return v.ID == u.ID && v.Type == u.Type && v.Released.Equal(u.Released)
}

// String returns v.ID.
func (v Version) String() string {
	return v.ID
}

///////////////////

// Type represents the release type of a version.
type Type string

const (
	Release  Type = "release"   // Ordinary release
	Snapshot Type = "snapshot"  // Development snapshot
	Alpha    Type = "old_alpha" // An alpha version
	Beta     Type = "old_beta"  // A beta version
)

// String returns a description of the version type meant for humans:
//	Release.String()  = "release"
//	Snapshot.String() = "snapshot"
//	Alpha.String()    = "alpha"
// 	Beta.String()     = "beta"
//	Type("").String() = "???" // Zero value
// Users SHOULD NOT rely on the results of String for other version types than
// the ones specified above. The default description for an unknown Type X is
// string(X), but as future versions of this library becomes aware of new
// Mojang-introduced types, for the previous unknown Type A, A.String() MAY
// change to differ from string(A). Once String has been specified for a given
// Type A, the return value of A.String() won't change further.
func (t Type) String() string {
	switch t {
	case Release:
		return "release"
	case Snapshot:
		return "snapshot"
	case Alpha:
		return "alpha"
	case Beta:
		return "beta"
	case "":
		return "???"
	default:
		return string(t)
	}
}

///////////////////

var errUnknownFormat = errors.New("unknown JSON data format")

type errHttpStatus int

func (e errHttpStatus) Error() string {
	code := int(e)
	return fmt.Sprintf("%d %s", code, http.StatusText(code))
}

///////////////////

var client = &http.Client{}

// Fetch Minecraft version JSON and parse it into a map hierarchy
func fetchJSON(ctx context.Context) (map[string]interface{}, error) {
	// Fetch JSON
	req, _ := http.NewRequest("GET", versionsURL, nil) // Error only occurs if versionsURL is bad
	req = req.WithContext(ctx)
	resp, err := client.Do(req) // TODO: Cache response and perform conditional requests
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, uerr("Get", errHttpStatus(resp.StatusCode))
	}

	// Decode JSON
	var j map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&j)
	if err != nil {
		return nil, uerr("Parse", err)
	}

	return j, nil
}

func initialize(l *Listing, m map[string]interface{}) (err error) {
	defer func() { // If JSON data isn't structured as expected
		if r := recover(); r != nil {
			err = uerr("Parse", errUnknownFormat)
		}
	}()

	l.Versions = make(map[string]Version)

	latest := m["latest"].(map[string]interface{})
	l.Latest.Snapshot = latest["snapshot"].(string)
	l.Latest.Release = latest["release"].(string)

	vm := l.Versions
	for _, v := range m["versions"].([]interface{}) {
		var vers Version
		buildVersion(v.(map[string]interface{}), &vers)
		vm[vers.ID] = vers
	}

	return
}

func buildVersion(m map[string]interface{}, v *Version) {
	v.ID = m["id"].(string)
	v.Released = parseTime(m["releaseTime"].(string))
	v.Type = Type(m["type"].(string))
}

func parseTime(t string) time.Time {
	const timeFormat = "2006-01-02T15:04:05-07:00"
	tm, _ := time.Parse(timeFormat, t)
	return tm
}

func uerr(op string, err error) *url.Error {
	return &url.Error{
		Op:  op,
		URL: versionsURL,
		Err: err,
	}
}
