document.querySelector("#innentuere").addEventListener("click", async (e) => {
    /** @type {HTMLButtonElement} */ const button = e.currentTarget;
    button.disabled = true;
    const response = await fetch("/open/innerdoor", {
        method: "POST",
    });
    button.disabled = false;

    if (response.ok) {
        return;
    }
    if (response.status === 425) {
        handleTooEarly()
        return;
    }
    if (response.status === 401) {
        handleUnauthorized()
        return;
    }
    handleUnknownError()
});

document.querySelector("#aussentuere").addEventListener("click", async (e) => {
    /** @type {HTMLButtonElement} */ const button = e.currentTarget;
    button.disabled = true;
    const response = await fetch("/open/outerdoor", {
        method: "POST",
    });
    button.disabled = false;

    if (response.ok) {
        return;
    }
    if (response.status === 425) {
        handleTooEarly()
        return;
    }
    if (response.status === 401) {
        handleUnauthorized()
        return;
    }
    handleUnknownError()
});

function handleTooEarly() {
    addToastItem("Too fast! Wait at least 10 seconds until you press again.");
}

function handleUnauthorized() {
    addToastItem("Unauthorized! Will reload in 5 seconds.", "error");

    setTimeout(() => {
        location.reload();
    }, 5000);
}

async function handleUnknownError(/** @type {Response} */ response) {
    addToastItem(`Unknown error: ${response.status} - ${await response.text()}! Will reload in 5 seconds.`, "error");

    setTimeout(() => {
        location.reload();
    }, 5000);
}

function addToastItem(/** @type {string} */ text, /** @type {string} */ type = "warning") {
    const toastItem = document.createElement("div");
    toastItem.classList.add("toast-item", `toast-item-${type}`);
    toastItem.innerText = text;

    document.querySelector(".toast").appendChild(toastItem);
    setTimeout(() => {
        toastItem.remove();
    }, 5000);
}
