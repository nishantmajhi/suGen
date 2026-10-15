document.addEventListener("DOMContentLoaded", () => {
    const passphraseLabel = document.getElementById("passphrase_range_label");
    const passwordLabel = document.getElementById("password_range_label");

    const originalPassphraseText = passphraseLabel.textContent.split(":")[0];
    const originalPasswordText = passwordLabel.textContent.split(":")[0];

    document.getElementById("passphrase_phrase_range").addEventListener("input", (event) => {
        updatePassphraseWordCountLabel(event.target);
    });

    document.getElementById("password_range").addEventListener("input", (event) => {
        passwordLabel.textContent = `${originalPasswordText}: ${event.target.value}`;
    });

    document.getElementById("numbers_in_password").addEventListener("change", syncMinimumInputs);
    document.getElementById("special_characters_in_password").addEventListener("change", syncMinimumInputs);

    document.getElementById("passphrase_style").addEventListener("change", () => {
        updatePassphrase();
    });
});

// On back/forward, the browser restores the form's ticks and values after
// DOMContentLoaded and before pageshow, without firing change events. So
// the first read of the form waits for pageshow. A page restored whole from
// the back/forward cache still has its state, and is left alone.
window.addEventListener("pageshow", (event) => {
    if (event.persisted) return;
    syncMinimumInputs();
    initialize();
});

// Each minimum is enabled exactly when its box is ticked: set from the box,
// not toggled, so a restored tick cannot leave it backwards.
function syncMinimumInputs() {
    document.getElementById("minimum_digits_in_password").disabled =
        !document.getElementById("numbers_in_password").checked;
    document.getElementById("minimum_special_characters_in_password").disabled =
        !document.getElementById("special_characters_in_password").checked;
}

// The slider's HTML min/max stay at 1-5 for both the "words" and
// "adjectiveNoun" styles, but a single word never makes sense for either
// style (a lone word isn't a passphrase, and "adjectiveNoun" always
// needs at least 1 adjective plus 1 noun) so we bounce the value back up
// to 2 in real time if the user drags it down to 1.
const MIN_FUNCTIONAL_WORD_COUNT = 2;

function updatePassphraseWordCountLabel(rangeInput) {
    const passphraseLabel = document.getElementById("passphrase_range_label");
    const originalPassphraseText = passphraseLabel.textContent.split(":")[0];

    if (Number(rangeInput.value) < MIN_FUNCTIONAL_WORD_COUNT) {
        rangeInput.value = String(MIN_FUNCTIONAL_WORD_COUNT);
    }

    passphraseLabel.textContent = `${originalPassphraseText}: ${rangeInput.value}`;
}

function initialize() {
    const passphraseLabel = document.getElementById("passphrase_range_label");
    const passwordLabel = document.getElementById("password_range_label");

    const originalPassphraseText = passphraseLabel.textContent.split(":")[0];
    const originalPasswordText = passwordLabel.textContent.split(":")[0];

    document.getElementById("passphrase_phrase_range").value = "2";
    passphraseLabel.textContent = `${originalPassphraseText}: 2`;
    // The length slider is not reset here, so the label reads its value.
    passwordLabel.textContent = `${originalPasswordText}: ${document.getElementById("password_range").value}`;

    updatePassphrase();
    updatePassword();
}
