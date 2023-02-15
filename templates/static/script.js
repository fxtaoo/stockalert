url = window.location.href.split('?')[0]
title = document.getElementById("title")
title.setAttribute("href", url)

async function updateResult() {
    const response = await fetch(url + 'updateresult')
    if (response.ok) {
        const result = await response.text()
        alert(result)
    } else {
        alert('updateResult 请求错误')
    }
}
