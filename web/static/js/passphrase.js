// The passphrase in the box, or null while the box shows anything else.
let shownPassphrase = null;
let latestPassphraseRequest = 0;

async function updatePassphrase() {
  const request = ++latestPassphraseRequest;
  const data = new URLSearchParams();
  data.append("style", document.getElementById("passphrase_style").value);
  data.append(
    "capitalizeWords",
    document.getElementById("capitalize_words_in_passphrase").checked
  );
  data.append(
    "includeNumbers",
    document.getElementById("include_numbers_in_passphrase").checked
  );
  data.append(
    "totalWords",
    document.getElementById("passphrase_phrase_range").value
  );

  try {
    const response = await fetch(
      `${window.location.origin}/passphrase/?${data.toString()}`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
        },
      }
    );

    if (response.ok) {
      const result = await response.json();
      // An answer to an earlier request must not replace a newer one.
      if (request !== latestPassphraseRequest) return;
      shownPassphrase = typeof result.passphrase === "string" ? result.passphrase : null;
      document.getElementById("passphrase").textContent = shownPassphrase ?? result.error;
    }
  } catch (error) {
    console.error("Network error:", error.message);
  }
}
