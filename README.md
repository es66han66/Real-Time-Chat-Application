## Steps to run application<br>
1. Run ```docker-compose up --build```.<br>
2. There are 2 sample clients(users who can chat with each other) file named ```client1.html``` and ```client2.html```. Both of these can be opened on a browser and as soon as we open any of them, we would get a log in our server with the name of user who connected to our server.<br>
3. We can send the message by inputting it on Type Message input box and pressing Send button. We should be able to see the message on 2nd client and vice versa.<br>
4. Similarly if we don't connect other client and keep sending messages from only 1, the messages will be enqueued and finally when the disconnected client connects, they will get all the enqueued messages.
