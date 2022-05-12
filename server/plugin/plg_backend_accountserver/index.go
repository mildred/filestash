package plg_backend_local

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	. "github.com/mickael-kerjean/filestash/server/common"
)

const typeName = "accountserver"

func init() {
	Backend.Register(typeName, &AccountServer{})
}

type AccountServer struct {
	path string
}

func checkParamsWithConfig(params map[string]string) error {
	if typeName != params["type"] {
		return ErrNotValid
	}

	for i := 0; i < len(Config.Conn); i++ {
		cfg := Config.Conn[i]
		if cfg["type"] != typeName {
			continue
		}

		valid := true

		// Check that all passed values are according to config
		for param, param_value := range params {
			config_value, ok := cfg[param]
			if ok && config_value != param_value {
				valid = false
				Log.Debug("[%s] Mismatch value %v, passed: %+v, config: %+v", typeName, param, param_value, config_value)
				break
			}
		}

		if !valid {
			continue
		}

		// Check that all values defined in config are passed with the same value
		for param, config_value := range cfg {
			if param == "advanced" || param == "label" {
				continue
			}
			param_value, ok := params[param]
			if !ok || config_value != param_value {
				valid = false
				Log.Debug("[%s] Missing value %v, passed: %+v, config: %+v", typeName, param, param_value, config_value)
				break
			}
		}

		if !valid {
			continue
		}

		return nil
	}

	return ErrNotValid
}

func (this AccountServer) Init(params map[string]string, app *App) (IBackend, error) {
	err := checkParamsWithConfig(params)
	if err != nil {
		return this, err
	}

	p := struct {
		username      string
		password      string
		path_template string
		url           string
	}{
		params["username"],
		params["password"],
		params["path_template"],
		params["url"],
	}

	res, err := http.PostForm(p.url, url.Values{
		"req":  []string{"checkauth"},
		"user": []string{p.username},
		"pass": []string{p.password},
	})
	if err != nil {
		return this, fmt.Errorf("cannot query login info, %v", err)
	}

	var authenticated bool

	defer res.Body.Close()

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return this, fmt.Errorf("cannot read login info, %v", err)
	}

	err = json.Unmarshal(resData, &authenticated)
	if err != nil {
		return this, fmt.Errorf("cannot decode login info %+v, %v", string(resData), err)
	}

	if !authenticated {
		return this, ErrAuthenticationFailed
	}

	root_path := strings.ReplaceAll(p.path_template, "%{username}", p.username)
	Log.Info("[accountserver] Authenticate %v to %v", p.username, root_path)

	return &AccountServer{path: root_path}, nil
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
				Description: "Username",
				Placeholder: "Username",
			},
			{
				Name:        "password",
				Type:        "password",
				Description: "Password",
				Placeholder: "Password",
			},
			{
				Name:        "advanced",
				Type:        "enable",
				Description: "Advanced",
				Placeholder: "Advanced",
				Target:      []string{"path_template", "url"},
			},
			{
				Id:          "path_template",
				Name:        "path_template",
				Description: "Path",
				Type:        "text",
				Placeholder: "/tmp/%{username}",
			},
			{
				Id:          "url",
				Name:        "url",
				Description: "Accountserver HTTP API Address",
				Type:        "text",
				Placeholder: "http://accountserver:8000",
			},
		},
	}
}

func (this AccountServer) resolve(p string) string {
	return path.Join(this.path, path.Join("/", p))
}

func (this AccountServer) Home() (string, error) {
	home := "/"
	return home, nil
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
