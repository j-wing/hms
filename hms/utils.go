package hms

import (
	"fmt"
	"html/template"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"google.golang.org/appengine"
	"google.golang.org/appengine/user"
)

const ALPHABET = "BV-XDyI4JLQ06KYH8G3OZ1FE7U9C25RSMNWTPA"

func IsLowercase(a byte) bool {
	return 97 <= a && a <= 122
}
func ShortURLEncode(n int64) string {
	base := int64(len(ALPHABET))
	num_digits := int64(1 + math.Floor((math.Log(float64(n)) / math.Log(float64(base)))))

	chars := make([]byte, num_digits)

	var remainder int64
	i := int64(0)
	for n > 0 {
		remainder = n % base
		chars[num_digits-i-1] = ALPHABET[remainder]
		n /= base
		i++
	}

	return string(chars)
}

func ShortURLDecode(s string) int64 {
	base := len(ALPHABET)

	var result int64 = 0
	var alphabet_index int64
	i := 0
	for _, char := range s {
		alphabet_index = int64(strings.IndexRune(ALPHABET, char))
		if alphabet_index == -1 {
			return -1
		}

		power := float64(len(s) - i - 1)
		result += int64(math.Pow(float64(base), power)) * alphabet_index
		i++
	}
	return result
}

func getTemplateBaseDir() string {
	if _, err := os.Stat("./tmpl"); err == nil {
		return "./tmpl"
	} else {
		return "../tmpl"
	}
}

func isAuthorizedUser(user user.User) bool {
	_, ok := ALLOWED_EMAILS[user.Email]
	return ok
}

func handleUserAuth(w http.ResponseWriter, r *http.Request) (*user.User, bool) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		loginUrl, _ := user.LoginURL(c, "/")
		http.Redirect(w, r, loginUrl, http.StatusFound)
		return nil, true
	} else if !isAuthorizedUser(*u) {
		return nil, false
	}
	return u, true
}

func getTemplate(path string) (*template.Template, error) {
	return template.ParseFiles(templateBaseDir + "/" + path)
}

func getErrorTemplate(e *appError) (*template.Template, error) {
	return getTemplate(fmt.Sprintf("errors/%d.html", e.Code))
}

func randomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ALPHABET[rand.Intn(len(ALPHABET))]

	}
	return string(b)

}

// returns whether path is suitable as a short link path
func isValidPath(path string) bool {
	return !(strings.Contains(path, "/"))
}

//func GetRouteHandler(routes map[string])
