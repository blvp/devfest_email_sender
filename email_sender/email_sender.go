package email_sender

import (
	"gopkg.in/gomail.v2"
	"crypto/tls"
	"log"
	"time"
)

type SenderConfig struct {
	Login     string `yaml:"login"`
	Password  string `yaml:"password"`
	Subject   string `yaml:"subject"`
	Host      string `yaml:"host"`
	Port      int `yaml:"port"`
	EnableSSL bool `yaml:"enableSsl"`
	From      string `yaml:"from"`

}

type User struct {
	UserName string `csv:"name"`
	Email    string `csv:"email"`
}

type UserEmail struct {
	Id      int
	User    *User
	Message string
	Status  string
}

type EmailSender struct {
	UserEmails  []*UserEmail
	MailChannel chan *UserEmail
	Subject     string
	From        string
}




func (es EmailSender) SendFailedOrCreated() {
	for _, email := range es.UserEmails {
		if email.Status == "failed" || email.Status == "created" {
			es.MailChannel <- email
		}
	}
}

func (es *EmailSender) CleanQueue() {
	es.UserEmails = es.UserEmails[:0]
}

func NewEmailSender(sc SenderConfig) *EmailSender {
	ch := make(chan *UserEmail)
	es := &EmailSender{
		Subject: sc.Subject,
		MailChannel: ch,
		From: sc.From,
	}
	go func() {
		d := gomail.NewDialer(sc.Host, sc.Port, sc.Login, sc.Password)
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		d.SSL = sc.EnableSSL
		var s gomail.SendCloser
		var err error
		open := false
		for {
			select {
			case m, ok := <-ch:
				if !ok {
					return
				}
				if !open {
					if s, err = d.Dial(); err != nil {
						panic(err)
					}
					open = true
				}
				m.Status = "proccessed"
				if err := gomail.Send(s, es.CreateMessage(m)); err != nil {
					log.Print(err)
					log.Println(m.User)
					m.Status = "failed"
				} else {
					m.Status = "sended"
				}
			// Close the connection to the SMTP server if no email was sent in
			// the last 30 seconds.
			case <-time.After(30 * time.Second):
				if open {
					if err := s.Close(); err != nil {
						panic(err)
					}
					open = false
				}
			}
		}
	}()

	return es
}
func (es EmailSender) Close() {
	close(es.MailChannel)
}

func (es EmailSender) CreateMessage(email *UserEmail) *gomail.Message {
	m := gomail.NewMessage()
	m.SetHeader("From", es.From)
	m.SetHeader("To", email.User.Email)
	m.SetHeader("Subject", es.Subject)
	m.SetBody("text/html", email.Message)
	return m
}
