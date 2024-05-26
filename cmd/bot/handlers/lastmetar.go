package handlers

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	dg "github.com/bwmarrin/discordgo"
)

type scanner struct {
	reader io.ReaderAt
	pos    int64
	err    error
	buf    []byte
}

func (s *scanner) read() {
	const chnksz = 1024

	if s.pos == 0 {
		s.err = io.EOF
		return
	}

	size := int64(chnksz)
	if size > s.pos {
		size = s.pos
	}
	s.pos -= size
	buf := make([]byte, size, size+int64(len(s.buf)))

	_, s.err = s.reader.ReadAt(buf, s.pos)
	if s.err == nil {
		s.buf = append(buf, s.buf...)
	}
}

func (s *scanner) Line() (string, error) {
	if s.err != nil {
		return "", s.err
	}

	for {
		lineStart := bytes.LastIndexByte(s.buf, '\n')

		if lineStart >= 0 {
			var line string
			line = strings.TrimSpace(string(s.buf[lineStart+1:]))
			s.buf = s.buf[:lineStart]
			return line, nil
		}

		s.read()
		if s.err != nil {
			if s.err == io.EOF {
				if len(s.buf) > 0 {
					return strings.TrimSpace(string(s.buf)), nil
				}
			}
			return "", s.err
		}
	}
}

var reMETAR = regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} METAR: (?P<metar>.*)`)

func LastMETAR(s *dg.Session, i *dg.InteractionCreate, rwLogPath string) {
	const command = `/last-metar`
	log.Println(command, "called")
	defer timeCommand(command)()

	if ok := verifyCaller(s, i, command, false); !ok {
		return
	}

	f, err := os.Open(rwLogPath)
	if err != nil {
		log.Printf("Unable to open Real Weather log file: %v", err)
		somethingWentWrong(s, i)
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	var metar string
	for sc.Scan() {
		if match := reMETAR.FindStringSubmatch(sc.Text()); len(match) == 2 {
			metar = match[1]
		}
	}

	if metar == "" {
		log.Println("Unable to locate a METAR in your Real Weather log file. Is your configuration correct?")
		s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
			Type: dg.InteractionResponseChannelMessageWithSource,
			Data: &dg.InteractionResponseData{
				Content: "Sorry, a METAR could not be found. Check your log file for more info.",
			},
		})
	}

	s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
		Type: dg.InteractionResponseChannelMessageWithSource,
		Data: &dg.InteractionResponseData{
			Content: metar,
		},
	})
}
