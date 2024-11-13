// async function run() {
//     const res = await fetch("https://discord.com/api/webhooks/", {
//         method: "POST",
//         headers: {
//             "Content-Type": "application/json"
//         },
//         body: JSON.stringify({
//             content: `Hey everyone! leez just posted on her main! Go check it out!
//             https://www.instagram.com/p/DCM3K69TJ39/`,
//         })
//     })
//
//     console.log(res)
// }
//
// run()
export async function POST(request: Request) {
    console.log(request.headers)
    const body = await request.json()

    console.log(JSON.stringify(body))
    return new Response(`Hello from ${process.env.VERCEL_REGION}`);
}
