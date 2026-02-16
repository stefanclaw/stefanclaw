package fetch

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// URLPattern matches http and https URLs in user messages.
var URLPattern = regexp.MustCompile(`https?://[^\s)<>]+`)

// AugmentWithWebContent detects URLs in userInput, fetches them, and returns
// the input with fetched web content appended. If no URLs are found or all
// fetches fail, the original input is returned unchanged.
func AugmentWithWebContent(ctx context.Context, c *Client, userInput string) string {
	urls := URLPattern.FindAllString(userInput, 3)
	if len(urls) == 0 {
		return userInput
	}

	var fetched []string
	for _, u := range urls {
		content, err := c.Fetch(ctx, u)
		if err == nil && content != "" {
			fetched = append(fetched, fmt.Sprintf("<webpage url=\"%s\">\n%s\n</webpage>", u, content))
		}
	}
	if len(fetched) == 0 {
		return userInput
	}

	return userInput + "\n\n" +
		"The following web page content was fetched for reference. " +
		"Use it to answer my question above — do NOT reproduce or summarize the raw page. " +
		"Focus only on the parts relevant to what I asked.\n\n" +
		strings.Join(fetched, "\n\n")
}
