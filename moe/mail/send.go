package mail

import (
	"github.com/labstack/gommon/log"
	"net/smtp"
)

type EmailMessage struct {
	Recipient []string
	Subject   string
	Body      string
}

func (e *Email) Send(to string) {
	message := &EmailMessage{
		Recipient: []string{"to"},
		Subject:   "来自SMOE的回复",
		Body:      "感谢您的留言，博主会在一周内回复您",
	}
	// 发送邮件
	c, err := smtp.NewClient(e.Conn, "smtp.gmail.com")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if err := c.Auth(e.Auth); err != nil {
		log.Fatal(err)
	}

	if err := c.Mail(e.user); err != nil {
		log.Fatal(err)
	}

	for _, addr := range message.Recipient {
		if err := c.Rcpt(addr); err != nil {
			log.Fatal(err)
		}
	}

	w, err := c.Data()
	if err != nil {
		log.Fatal(err)
	}

	_, err = w.Write([]byte(message.Body))
	if err != nil {
		log.Fatal(err)
	}

	err = w.Close()
	if err != nil {
		log.Fatal(err)
	}
}
