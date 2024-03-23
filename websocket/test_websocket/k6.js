import { WebSocket } from 'k6/experimental/websockets';
import { sleep } from 'k6';
export default function () {
    const ws = new WebSocket('ws://154.83.13.70:8081/ws');

    ws.onopen = () => {
        ws.send('1111');

    };
    ws.onmessage = (data) => {
        console.log('a message received');
        console.log(data);
        ws.close(); // 这边 close 服务端就会 err !=nil
    };

}


