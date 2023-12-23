const API_STORAGE_KEY = "rtnlapistoragekey";

(function() {

  // Handle responses from the login form.
  document.body.addEventListener("htmx:afterRequest", function(e) {
    if (e.detail.failed) {
      // Handle error from the server
      let message = "could not complete request, please try again later"
      if (e.detail.xhr.response) {
        const data = JSON.parse(e.detail.xhr.response);
        message = data.error;
      }

      let elem = document.getElementById("login-error")
      elem.innerText = message;
      elem.classList.remove("hidden");
      return
    }

    // Otherwise the API key is valid and can be set on local storage.
    const data = JSON.parse(e.detail.xhr.response);
    window.localStorage.setItem(API_STORAGE_KEY, data.apikey);
    window.location.href = "/"
  })

})();