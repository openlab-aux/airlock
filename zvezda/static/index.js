document.querySelector("#innentuere").addEventListener('click', async (e) => {
    const button = e.currentTarget
    button.disabled = true
    const response = await fetch('/open/innerdoor', {
        method: 'POST',
    })
    button.disabled = false

    if (response.ok) {
        return
    }
    console.log(response.status)

    if (response.status === 425) {
        const toastItem = document.createElement("div")
        toastItem.classList.add("toast-item")
        toastItem.innerText = "Too fast! Wait at least 10 seconds until you press again."

        document.querySelector(".toast").appendChild(toastItem)
        setTimeout(() => {
            toastItem.remove()
        }, 5000)
        return
    }

    const toastItem = document.createElement("div")
    toastItem.classList.add("toast-item")
    toastItem.innerText = "Too fast! Wait at least 10 seconds until you press again."

    document.querySelector(".toast").appendChild(toastItem)
    setTimeout(() => {
        toastItem.remove()
    }, 5000)

})

document.querySelector("#aussentuere").addEventListener('click', async (e) => {
    const button = e.currentTarget
    button.disabled = true
    await fetch('/open/outerdoor', {
        method: 'POST',
    })
    button.disabled = false
})
