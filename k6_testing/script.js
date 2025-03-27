import ws from 'k6/ws';
import { check, sleep } from 'k6';

export let options = {
    stages: [
        { duration: '10s', target: 100 }
    ],
};

export default function () {
    let playerId = String(Math.floor(Math.random() * 1000));
    const url = `ws://192.168.4.42:4002/ws?playerId=${playerId}`;
    const params = {
        tags: { my_tag: 'testing' },
    };

    const res = ws.connect(url, params, function (socket) {

        socket.on('open', () => {

            console.log('Connected to the WebSocket server');

            socket.setInterval(function timeout() {
                socket.send('p')
                console.log('Pinging every 1sec (setInterval test)');
            }, 1000);

            let name = `test_player${playerId}`;

            let snakeColors = {
                head: 'green',
                body: 'yellow',
                eyes: 'black',
            };

            socket.send(JSON.stringify({
                event: "newPlayer",
                player: { name, id: playerId, colours: { head: snakeColors.head, body: snakeColors.body, eyes: snakeColors.eyes } }
            }));

            sleep(1)

            socket.send(JSON.stringify({ event: 'startGame' }));
        });



        socket.on('message', function (msg) {
            if (!msg.startsWith('{')) {
                if (msg === 'p') {

                    check(msg, {
                        'Pong received': (m) => m === 'p'
                    });
                }
                return
            }
            const parsed = JSON.parse(msg)


            if (parsed.event === 'snake_update') {
                check(parsed, {
                    'Game update received': (parsed) => parsed.event === 'snake_update',
                });
                return
            }
        });

        socket.on('close', () => {

            console.log('WebSocket connection closed');
        });

    });


    check(res, {
        'WebSocket connection established': (r) => r.status === 101,
    });
}

