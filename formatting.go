package main

import (
	"fmt"
	"regexp"
	"strings"

	"maunium.net/go/mautrix/format"

	"github.com/Food-to-Share/bridge/types"
)

var italicRegex = regexp.MustCompile("([\\s>~*]|^)_(.+?)_([^a-zA-Z\\d]|$)")
var boldRegex = regexp.MustCompile("([\\s>_~]|^)\\*(.+?)\\*([^a-zA-Z\\d]|$)")
var strikethroughRegex = regexp.MustCompile("([\\s>_*]|^)~(.+?)~([^a-zA-Z\\d]|$)")
var codeBlockRegex = regexp.MustCompile("```(?:.|\n)+?```")
var mentionRegex = regexp.MustCompile("@[0-9]+")

type Formatter struct {
	bridge *Bridge

	matrixHTMLParser *format.HTMLParser

	waReplString   map[*regexp.Regexp]string
	waReplFunc     map[*regexp.Regexp]func(string) string
	waReplFuncText map[*regexp.Regexp]func(string) string
}

func NewFormatter(bridge *Bridge) *Formatter {
	formatter := &Formatter{
		bridge: bridge,
		matrixHTMLParser: &format.HTMLParser{
			TabsToSpaces: 4,
			Newline:      "\n",

			PillConverter: func(mxid, eventID string) string {
				if mxid[0] == '@' {
					puppet := bridge.GetPuppetByMXID(mxid)
					fmt.Println(mxid, puppet)
					if puppet != nil {
						return "@" + mxid()
					}
				}
				return mxid
			},
			BoldConverter: func(text string) string {
				return fmt.Sprintf("*%s*", text)
			},
			ItalicConverter: func(text string) string {
				return fmt.Sprintf("_%s_", text)
			},
			StrikethroughConverter: func(text string) string {
				return fmt.Sprintf("~%s~", text)
			},
			MonospaceConverter: func(text string) string {
				return fmt.Sprintf("```%s```", text)
			},
			MonospaceBlockConverter: func(text string) string {
				return fmt.Sprintf("```%s```", text)
			},
		},
		waReplString: map[*regexp.Regexp]string{
			italicRegex:        "$1<em>$2</em>$3",
			boldRegex:          "$1<strong>$2</strong>$3",
			strikethroughRegex: "$1<del>$2</del>$3",
		},
	}
	formatter.waReplFunc = map[*regexp.Regexp]func(string) string{
		codeBlockRegex: func(str string) string {
			str = str[3 : len(str)-3]
			if strings.ContainsRune(str, '\n') {
				return fmt.Sprintf("<pre><code>%s</code></pre>", str)
			}
			return fmt.Sprintf("<code>%s</code>", str)
		},
		mentionRegex: func(str string) string {
			mxid, displayname := formatter.getMatrixInfoByJID(str[1:])
			return fmt.Sprintf(`<a href="https://matrix.to/#/%s">%s</a>`, mxid, displayname)
		},
	}
	formatter.waReplFuncText = map[*regexp.Regexp]func(string) string{
		mentionRegex: func(str string) string {
			_, displayname := formatter.getMatrixInfoByJID(str[1:])
			return displayname
		},
	}
	return formatter
}

func (formatter *Formatter) getMatrixInfoByJID(jid types.AppID) (mxid, displayname string) {
	if user := formatter.bridge.GetUserByJID(jid); user != nil {
		mxid = user.MXID
		displayname = user.MXID
	} else if puppet := formatter.bridge.GetPuppetByJID(jid); puppet != nil {
		mxid = puppet.MXID
		displayname = puppet.Displayname
	}
	return
}

func (formatter *Formatter) ParseMatrix(html string) string {
	return formatter.matrixHTMLParser.Parse(html)
}
