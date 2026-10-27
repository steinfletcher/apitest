package apitest

import (
	"bytes"
	"encoding/json"
	htmlTemplate "html/template"
	"io"
	"net/http"
	"sort"
	"strings"
)

// formatBodyContent reads the body, replaces it so that it can be read again, and returns it formatted for
// display (JSON is indented) along with the raw content
func formatBodyContent(bodyReadCloser io.ReadCloser, replaceBody func(replacementBody io.ReadCloser)) (string, string, error) {
	if bodyReadCloser == nil {
		return "", "", nil
	}

	body, err := io.ReadAll(bodyReadCloser)
	if err != nil {
		return "", "", err
	}

	replaceBody(io.NopCloser(bytes.NewReader(body)))

	return indentJSON(body), string(body), nil
}

// formatMessageBody formats the body of a custom message for display
func formatMessageBody(body string) string {
	return indentJSON([]byte(body))
}

// indentJSON indents valid JSON and returns anything else unchanged
func indentJSON(body []byte) string {
	if json.Valid(body) {
		var indented bytes.Buffer
		// json.Indent cannot fail on a body that json.Valid accepted
		if err := json.Indent(&indented, body, "", "  "); err == nil {
			return indented.String()
		}
	}
	return string(body)
}

// highlightJSON returns the JSON document as HTML with each token wrapped in a span classed by its type.
// It reports false if the input is not valid JSON.
func highlightJSON(in string) (htmlTemplate.HTML, bool) {
	trimmed := strings.TrimSpace(in)
	if trimmed == "" || !json.Valid([]byte(trimmed)) {
		return "", false
	}

	var out strings.Builder
	span := func(class, text string) {
		out.WriteString(`<span class="j-`)
		out.WriteString(class)
		out.WriteString(`">`)
		out.WriteString(htmlTemplate.HTMLEscapeString(text))
		out.WriteString(`</span>`)
	}

	for i := 0; i < len(in); {
		c := in[i]
		switch {
		case c == '"':
			end := i + 1
			for end < len(in) && in[end] != '"' {
				if in[end] == '\\' {
					end++
				}
				end++
			}
			end++ // closing quote
			if end > len(in) {
				end = len(in)
			}
			class := "s"
			rest := strings.TrimLeft(in[end:], " \t\r\n")
			if strings.HasPrefix(rest, ":") {
				class = "k"
			}
			span(class, in[i:end])
			i = end
		case c == '-' || (c >= '0' && c <= '9'):
			end := i + 1
			for end < len(in) && strings.IndexByte("0123456789.eE+-", in[end]) >= 0 {
				end++
			}
			span("n", in[i:end])
			i = end
		case strings.HasPrefix(in[i:], "true"):
			span("b", "true")
			i += 4
		case strings.HasPrefix(in[i:], "false"):
			span("b", "false")
			i += 5
		case strings.HasPrefix(in[i:], "null"):
			span("z", "null")
			i += 4
		case strings.IndexByte("{}[],:", c) >= 0:
			span("p", string(c))
			i++
		default:
			out.WriteString(htmlTemplate.HTMLEscapeString(string(c)))
			i++
		}
	}
	return htmlTemplate.HTML(out.String()), true
}

// curlCommand returns a curl command that sends the request, for copying from the report
func curlCommand(req *http.Request, body string) string {
	url := req.URL.String()
	if !req.URL.IsAbs() {
		host := req.Host
		if host == "" {
			host = SystemUnderTestDefaultName
		}
		url = "http://" + host + req.URL.RequestURI()
	}

	parts := []string{"curl"}
	if req.Method != "" && req.Method != http.MethodGet {
		parts = append(parts, "-X", req.Method)
	}
	parts = append(parts, shellQuote(url))

	names := make([]string, 0, len(req.Header))
	for name := range req.Header {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		for _, value := range req.Header[name] {
			parts = append(parts, "-H", shellQuote(name+": "+value))
		}
	}
	if body != "" {
		parts = append(parts, "--data-raw", shellQuote(body))
	}
	return strings.Join(parts, " ")
}

// shellQuote quotes s for a POSIX shell
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
