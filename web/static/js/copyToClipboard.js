async function copyPassword() {
    const passwordCopyIcon = document.getElementById("copy_password");

    try {
        // Copy the password itself, never a message standing in for one.
        if (shownPassword === null) throw new Error("No password to copy");
        await copyTextToClipboard(shownPassword);
        animateSwapping(passwordCopyIcon, "&#128505;", "&#10064;");
        showNotification("Password copied to clipboard!");
    } catch (err) {
        showNotification("Failed to copy password.");
    }
}

async function copyPassphrase() {
    const passphraseCopyIcon = document.getElementById("copy_passphrase");

    try {
        if (shownPassphrase === null) throw new Error("No passphrase to copy");
        await copyTextToClipboard(shownPassphrase);
        animateSwapping(passphraseCopyIcon, "&#128505;", "&#10064;");
        showNotification("Passphrase copied to clipboard!");
    } catch (err) {
        showNotification("Failed to copy passphrase.");
    }
}

function animateSwapping(button, newIcon, originalIcon) {
    button.disabled = true;

    button.classList.add("fade");
    setTimeout(() => {
        button.innerHTML = newIcon;
        button.classList.remove("fade");
    }, 150);

    setTimeout(() => {
        button.classList.add("fade");
        setTimeout(() => {
            button.innerHTML = originalIcon;
            button.classList.remove("fade");
            button.disabled = false;
        }, 150);
    }, 1500);
}

async function copyTextToClipboard(text) {
    if (navigator.clipboard && window.isSecureContext) {
        try {
            await navigator.clipboard.writeText(text);
        } catch (err) {
            console.error("Clipboard copy failed:", err);
            await fallbackCopyText(text);
        }
    } else {
        await fallbackCopyText(text);
    }
}

function fallbackCopyText(text) {
    return new Promise((resolve, reject) => {
        const textarea = document.createElement("textarea");
        textarea.value = text;
        textarea.style.position = "fixed";
        textarea.style.opacity = 0;
        document.body.appendChild(textarea);
        textarea.focus();
        textarea.select();

        try {
            const successful = document.execCommand("copy");
            document.body.removeChild(textarea);
            if (successful) {
                resolve();
            } else {
                reject(new Error("Fallback: Copy command failed"));
            }
        } catch (err) {
            document.body.removeChild(textarea);
            reject(err);
        }
    });
}