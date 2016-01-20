package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/bobbytables/slacker"
)

// TODO: handle typing notifications

var SlackToken = flag.String("slack-token", "", "Slack Token")
var SlackURL = flag.String("slack-url", "", "Slack URL (may be empty)")
var Channel = flag.String("channel", "general", "The name of the slack channel")
var Users = flag.String("users", "", "Only consider the specified users, separated by commas, rather than all users in the channel")

type StandupRunner struct {
	Channel *slacker.Channel
	Users   []*slacker.User
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Shuffle(a []*slacker.User) {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

func main() {
	flag.Parse()

	c := slacker.NewAPIClient(*SlackToken, *SlackURL)
	rtmStart, err := c.RTMStart()
	if err != nil {
		panic(err)
	}

	sr := StandupRunner{}

	channels, err := c.ChannelsList()
	if err != nil {
		panic(err)
	}
	for _, channel := range channels {
		if channel.Name == *Channel {
			sr.Channel = channel
		}
	}
	log.Printf("channel is: %s (%s)", sr.Channel.Name, sr.Channel.ID)

	authBuf, err := c.RunMethod("auth.test")
	if err != nil {
		panic(err)
	}
	var auth struct {
		UserID string `json:"user_id"`
	}
	json.Unmarshal(authBuf, &auth)

	var onlyUsers []string
	if *Users != "" {
		onlyUsers = strings.Split(*Users, ",")
	}

	users, err := c.UsersList()
	if err != nil {
		panic(err)
	}

	if onlyUsers != nil {
		for _, user := range users {
			for _, userName := range onlyUsers {
				if userName == user.Name {
					sr.Users = append(sr.Users, user)
				}
			}
		}
	} else {
		for _, user := range users {
			for _, userID := range sr.Channel.Members {
				if userID == auth.UserID {
					continue
				}
				if userID == user.ID {
					sr.Users = append(sr.Users, user)
				}
			}
		}
	}

	broker := slacker.NewRTMBroker(rtmStart)
	broker.Connect()
	defer broker.Close()
	messages := make(chan *slacker.RTMMessage)

	go func() {
		for {
			event := <-broker.Events()
			if event.Type == "message" {
				msg, err := event.Message()
				if err != nil {
					panic(err)
				}

				if msg.Channel != sr.Channel.ID {
					continue
				}

				messages <- msg
			}
		}
	}()

	UserName := func(userID string) string {
		for _, user := range sr.Users {
			if user.ID == userID {
				return user.Name
			}
		}
		return userID
	}

	FriendlyName := func(user *slacker.User) string {
		if user.FirstName != "" {
			return user.FirstName
		}
		if user.LastName != "" {
			return user.LastName
		}
		return user.Name
	}

	Say := func(s string) {
		log.Printf("I said: %s", s)
		err := broker.Publish(slacker.RTMMessage{
			Type:    "message",
			Text:    s,
			Channel: sr.Channel.ID,
		})
		if err != nil {
			panic(err)
		}
	}

	Shuffle(sr.Users)

	log.Printf("ready for standup in #%s", sr.Channel.Name)
	log.Printf("Users will be:")
	for _, user := range sr.Users {
		log.Printf("%s (%s)", user.ID, user.Name)
	}
	typingTime := time.Second * 2

	Say("Hello @channel, it's *standup time*.")
	time.Sleep(typingTime)
	Say("I'll call on each of you one at a time. " +
		"When you are done with your update, send a message containing a " +
		"single period `.` and we'll move on to the next person. " +
		"If I call on someone who is not here, say `.` and I will move on.")

	countUsersResponding := 0
	for _, user := range sr.Users {
		time.Sleep(typingTime)
		Say(fmt.Sprintf("@%s, what have you got for us?", user.Name))

		responseCount := 0
		for {
			msg := <-messages
			log.Printf("%s said: %s", UserName(msg.User), msg.Text)

			if msg.Text == "." {
				if responseCount == 0 && msg.User != user.ID {
					Say(fmt.Sprintf("Got it. %s is not here. If that's wrong, you can chime in at the end.",
						FriendlyName(user)))
				} else {
					Say(fmt.Sprintf("Cool. Thanks, %s.", FriendlyName(user)))
				}
				break
			}

			if msg.User != user.ID {
				continue
			}

			// the user spoke. every time they speak they get another
			// four mins on the deadline
			responseCount++
			if msg.Text == "." {
				Say(fmt.Sprintf("Cool. Thanks, %s.", FriendlyName(user)))
				break
			}
		}
		if responseCount > 0 {
			countUsersResponding++
		}
	}

	time.Sleep(typingTime)
	Say("Thanks everybody! See you next time.")
	log.Printf("done with standup")
}

