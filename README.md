
# standupbot

This is a super simple slackbot to run standups. Run it on a cron, like:

```
30 10 * * 1-5 /bin/bash -c 'path/to/standupbot -slack-token=xoxb-0000000000-yyyyyyyyyyyyyyyyyy -channel=myproject 2>&1 | logger'
```

How a standup works:

- The bot says "Hello @channel, it's ​*standup time*​. I'll call on each of you one at a time. When you are done with your update, send a message containing a single period `.` and we'll move on to the next person. If I call on someone who is not here, say `.` and I will move on.
- For each person in the channel (or each person listed if you specify -users on the command line), it asks them "@alice, what have you got for us?". When anyone types a period (`.`) in a message by itself, the bot moves on.

Reference: https://xkcd.com/1319/

