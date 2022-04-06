package plg_backend_local

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	. "github.com/mickael-kerjean/filestash/server/common"
)

func init() {
	Backend.Register("accountserver", &AccountServer{})
}

type AccountServer struct {
	path string
}

func (this AccountServer) Init(params map[string]string, app *App) (IBackend, error) {
	p := struct {
		username      string
		password      string
		path          string
		accountserver string
	}{
		params["username"],
		params["password"],
		params["path"],
		params["accountserver"],
	}

	res, err := http.PostForm(p.accountserver, url.Values{
		"req":  []string{"checkauth"},
		"user": []string{p.username},
		"pass": []string{p.password},
	})
	if err != nil {
		return this, err
	}

	var authenticated bool

	err = json.NewDecoder(res.Body).Decode(&authenticated)
	if err != nil {
		return this, err
	}

	if !authenticated {
		return this, ErrAuthenticationFailed
	}

	return &AccountServer{path: strings.ReplaceAll(p.path, "%{username}", p.username)}, nil
}

func (this AccountServer) LoginForm() Form {
	return Form{
		Elmnts: []FormElement{
			{
				Name:  "type",
				Type:  "hidden",
				Value: "accountserver",
			},
			{
				Name:        "username",
				Type:        "text",
				Placeholder: "Username",
			},
			{
				Name:        "password",
				Type:        "password",
				Placeholder: "Password",
			},
			{
				Name:        "advanced",
				Type:        "enable",
				Placeholder: "Advanced",
				Target:      []string{"path", "accountserver"},
			},
			{
				Id:          "path",
				Name:        "path",
				Type:        "text",
				Placeholder: "/tmp/%{username}",
			},
			{
				Id:          "accountserver",
				Name:        "accountserver",
				Type:        "text",
				Placeholder: "accountserver:8001",
			},
		},
	}
}

func (this AccountServer) resolve(p string) string {
	return path.Join(this.path, path.Join("/", p))
}

func (this AccountServer) Ls(path string) ([]os.FileInfo, error) {
	f, err := os.Open(this.resolve(path))
	if err != nil {
		return nil, err
	}
	return f.Readdir(-1)
}

func (this AccountServer) Cat(path string) (io.ReadCloser, error) {
	return os.OpenFile(this.resolve(path), os.O_RDONLY, os.ModePerm)
}

func (this AccountServer) Mkdir(path string) error {
	return os.Mkdir(this.resolve(path), 0664)
}

func (this AccountServer) Rm(path string) error {
	return os.Remove(this.resolve(path))
}

func (this AccountServer) Mv(from, to string) error {
	return os.Rename(this.resolve(from), this.resolve(to))
}

func (this AccountServer) Save(path string, content io.Reader) error {
	f, err := os.OpenFile(this.resolve(path), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, content)
	return err
}

func (this AccountServer) Touch(path string) error {
	f, err := os.OpenFile(this.resolve(path), os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	if _, err = f.Write([]byte("")); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
