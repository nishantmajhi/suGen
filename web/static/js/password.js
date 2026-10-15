// The password in the box, or null while the box shows anything else.
let shownPassword = null;
let latestPasswordRequest = 0;

async function updatePassword() {
  const request = ++latestPasswordRequest;
  const data = new URLSearchParams();
  data.append(
    "mixedCase",
    document.getElementById("mixed_case_in_password").checked
  );
  data.append(
    "passwordLength",
    document.getElementById("password_range").value
  );
  data.append(
    "numbers",
    document.getElementById("numbers_in_password").checked
  );
  data.append(
    "minimumDigits",
    document.getElementById("minimum_digits_in_password").value
  );
  data.append(
    "specialCharacters",
    document.getElementById("special_characters_in_password").checked
  );
  data.append(
    "minimumSpecialCharacters",
    document.getElementById("minimum_special_characters_in_password").value
  );
  data.append(
    "avoidAmbiguousCharacters",
    document.getElementById("avoid_ambiguous_characters_in_password").checked
  );

  try {
    const response = await fetch(
      `${window.location.origin}/password/?${data.toString()}`,
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
      if (request !== latestPasswordRequest) return;
      // When the settings leave no room, the server says why in "error".
      shownPassword = typeof result.password === "string" ? result.password : null;
      document.getElementById("password").textContent = shownPassword ?? result.error;
    }
  } catch (error) {
    console.error("Network error:", error.message);
  }
}
