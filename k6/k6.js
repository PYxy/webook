import http from 'k6/http';

const url = "http://localhost:8080/hello"

export default function () {
    const data = {name: "Tom"}
    const okStatus = http.expectedStatuses(200)
    http.post(url, JSON.stringify(data), {
        headers: {'Content-Type': 'application/json'},
        // 传入一个预期的响应
        // 我预期它会返回 200
        responseCallback: okStatus
    })
}