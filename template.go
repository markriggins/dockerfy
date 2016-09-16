package main

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
)

type TemplateContext struct {
	env, secrets map[string]string
}

func GetEnvMap() map[string]string {
    env := make(map[string]string)
    for _, i := range os.Environ() {
        sep := strings.Index(i, "=")
        env[i[0:sep]] = i[sep+1:]
    }
    return env
}

//
// getenv template function
//
// '{{concat "P" "WD" | getenv}}' will print $PWD
//
func GetEnv(v string) string {
    return GetEnvMap()[v]
}

//
// .Env.VAR lookup from template context
//
func (c *TemplateContext) Env() map[string]string {
	if c.env != nil {
		return c.env
	}
	c.env = GetEnvMap()
	return c.env
}

//
// Make secrets available under .Secret.VARNAME in the TemplateContext
//
func (c *TemplateContext) Secret() map[string]string {
	if c.secrets != nil {
		return c.secrets
	}
	c.secrets = getSecrets()
	return c.secrets
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func contains(item map[string]string, key string) bool {
	if _, ok := item[key]; ok {
		return true
	}
	return false
}

func defaultValue(args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("default called with no values!")
	}

	if len(args) > 0 {
		if args[0] != nil {
			return args[0].(string), nil
		}
	}

	if len(args) > 1 {
		if args[1] == nil {
			return "", fmt.Errorf("default called with nil default value!")
		}

		if _, ok := args[1].(string); !ok {
			return "", fmt.Errorf("default is not a string value. hint: surround it w/ double quotes.")
		}

		return args[1].(string), nil
	}

	return "", fmt.Errorf("default called with no default value")
}

func parseUrl(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		log.Fatalf("unable to parse url %s: %s", rawurl, err)
	}
	return u
}

func add(arg1, arg2 int) int {
	return arg1 + arg2
}

func concat(args ...string) (s string) {
    s = ""
    for _, v := range args {
        s = s + v
    }
    return
}

func sequence(firstS, lastS string) []string {
    // return a sequence of strings from first (inclusive) to last (exlusive)
    // N(3,5) returns ["3", "4"]
    first, _ := strconv.Atoi(firstS)
    last, _ := strconv.Atoi(lastS)

    if last < first {
        last, first = 0, 0
    }
    sequence := make([]string, (last - first))
    for i := first; i < last; i++ {
        sequence[i] = strconv.Itoa(i)
    }
    return sequence
}

var funcMap = template.FuncMap{
        "contains": contains,
        "exists":   exists,
        "split":    strings.Split,
        "replace":  strings.Replace,
        "default":  defaultValue,
        "parseUrl": parseUrl,
        "atoi":     strconv.Atoi,
        "add":      add,
        "concat":   concat,
        "sequence": sequence,
        "N":        sequence,
        "getenv":   GetEnv,
    }
//
// Execute the string_template under the TemplateContext, and
// return the result as a string
//
func string_template_eval(string_template string) string {
	var result bytes.Buffer
    t := template.New("String Template").Funcs(funcMap)


	t, err := t.Parse(string_template)
	if err != nil {
		log.Fatalf("unable to parse template: %s", err)
	}

	err = t.Execute(&result, &TemplateContext{})
	if err != nil {
		log.Fatalf("template error: %s\n", err)
	}

	return result.String()
}

//
// Execute the template at templatePath under the TemplateContext and write it to destPath
//
func generateFile(templatePath, destPath string) bool {
	tmpl := template.New(filepath.Base(templatePath)).Funcs(funcMap)

	if len(delims) > 0 {
		tmpl = tmpl.Delims(delims[0], delims[1])
	}
	tmpl, err := tmpl.ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("unable to parse template: %s", err)
	}

	dest := os.Stdout
	if destPath != "" {
		dest, err = os.Create(destPath)
		if err != nil {
			log.Fatalf("unable to create %s", err)
		}
		defer dest.Close()
		if verboseFlag {
			log.Printf("Template %s --> %s\n", templatePath, destPath)
		}
	}

	err = tmpl.ExecuteTemplate(dest, filepath.Base(templatePath), &TemplateContext{})
	if err != nil {
		log.Fatalf("template error: %s\n", err)
	}

	if fi, err := os.Stat(destPath); err == nil {
		if err := dest.Chmod(fi.Mode()); err != nil {
			log.Fatalf("unable to chmod temp file: %s\n", err)
		}
		if err := dest.Chown(int(fi.Sys().(*syscall.Stat_t).Uid), int(fi.Sys().(*syscall.Stat_t).Gid)); err != nil {
			log.Fatalf("unable to chown temp file: %s\n", err)
		}
	}

	return true
}

