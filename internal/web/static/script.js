// Toggle the visibility of advanced options in the add form.
document.getElementById("toggleAdvanced").addEventListener("click", function(e) {
    e.preventDefault();
    var adv = document.getElementById("advancedOptions");
    if (adv.style.display === "none" || adv.style.display === "") {
      adv.style.display = "block";
      this.textContent = "Hide Advanced Options";
    } else {
      adv.style.display = "none";
      this.textContent = "Advanced Options";
    }
  });
  
  // Function to copy text and temporarily change the element text
  function copyText(el, text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text)
        .then(function() {
          var originalText = el.innerHTML;
          el.innerHTML = "Copied XML path to clipboard!";
          setTimeout(function() {
            el.innerHTML = originalText;
          }, 1000);
          console.log("Copied:", text);
        })
        .catch(function(err) {
          console.error("Error copying text:", err);
        });
    } else {
      var textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.style.position = "fixed";
      textarea.style.top = 0;
      textarea.style.left = 0;
      document.body.appendChild(textarea);
      textarea.focus();
      textarea.select();
      try {
        document.execCommand("copy");
        var originalText = el.innerHTML;
        el.innerHTML = "Copied XML path to clipboard!";
        setTimeout(function() {
          el.innerHTML = originalText;
        }, 1000);
        console.log("Fallback: Copied text:", text);
      } catch (err) {
        console.error("Fallback: Unable to copy", err);
      }
      document.body.removeChild(textarea);
    }
  }
  
  // Refresh the feed list from /feeds, then attach event listeners
  function refreshFeedList() {
    fetch("/feeds", {
      method: "GET",
      headers: { "X-Requested-With": "XMLHttpRequest" }
    })
      .then(response => response.text())
      .then(html => {
        document.getElementById("feedListWrapper").innerHTML = html;
        attachFeedListEventListeners();
      })
      .catch(error => {
        console.error("Error refreshing feed list:", error);
        document.getElementById("feedListWrapper").innerHTML =
          '<div class="message">Error loading feed list.</div>';
      });
  }
  
  // Attach event listeners to newly inserted feed list elements
  function attachFeedListEventListeners() {
    // "Edit Feed" button
    document.querySelectorAll('[data-role="edit-button"]').forEach(btn => {
      btn.addEventListener("click", function() {
        const feedKey = this.getAttribute("data-feedkey");
        toggleEdit(this, feedKey);
  
        setupEditFormChangeListeners(feedKey);
      });
    });
  
    // "Save Edit" button
    document.querySelectorAll('[data-role="save-edit"]').forEach(btn => {
      btn.addEventListener("click", function() {
        const feedKey = this.getAttribute("data-feedkey");
        confirmEdit(feedKey);
      });
    });
  
    // "Remove Feed" button
    document.querySelectorAll('[data-role="remove-feed"]').forEach(btn => {
      btn.addEventListener("click", function() {
        const feedKey = this.getAttribute("data-feedkey");
        removeFeed(this, feedKey);
      });
    });
  
    // "Copy XML path"
    document.querySelectorAll('[data-role="xml-button"]').forEach(span => {
      span.addEventListener("click", function() {
        const xmlurl = this.getAttribute("data-xmlurl");
        copyText(this, xmlurl);
      });
    });
  }
  
  /**
   * For each field in the edit form, compare its .value to its data-original attribute
   * and enable/disable the "Save Edit" button accordingly.
   */
  function setupEditFormChangeListeners(feedKey) {
    const prefix = feedKey + "-";
    const updateField   = document.getElementById(prefix + "update_period");
    const formatField   = document.getElementById(prefix + "format");
    const maxAgeField   = document.getElementById(prefix + "max_age");
    const keepLastField = document.getElementById(prefix + "clean_keep_last");
  
    // The Save button
    const saveBtn = document.querySelector(`[data-role="save-edit"][data-feedkey="${feedKey}"]`);
  
    function checkIfChanged() {
      let changed = false;
  
      if (updateField && updateField.value !== updateField.dataset.original) {
        changed = true;
      }
      if (formatField && formatField.value !== formatField.dataset.original) {
        changed = true;
      }
      if (maxAgeField && maxAgeField.value !== maxAgeField.dataset.original) {
        changed = true;
      }
      if (keepLastField && keepLastField.value !== keepLastField.dataset.original) {
        changed = true;
      }
  
      saveBtn.disabled = !changed;
    }
  
    // Listeners for each field
    if (updateField) {
      updateField.addEventListener("input", checkIfChanged);
    }
    if (formatField) {
      formatField.addEventListener("change", checkIfChanged);
    }
    if (maxAgeField) {
      maxAgeField.addEventListener("input", checkIfChanged);
    }
    if (keepLastField) {
      keepLastField.addEventListener("input", checkIfChanged);
    }
  }
  
  // Initial feed load
  refreshFeedList();
  
  // Reload button
  document.getElementById("reloadBtn").addEventListener("click", function() {
    var btn = this;
    btn.disabled = true;
    var originalText = btn.textContent;
    btn.textContent = "Reloading…";
    fetch("/reload", {
      method: "POST",
      headers: { "X-Requested-With": "XMLHttpRequest" }
    })
      .then(response => response.json())
      .then(data => {
        document.getElementById("messageContainer").innerHTML =
          '<div class="message">' + data.message + '</div>';
        btn.disabled = false;
        btn.textContent = originalText;
        refreshFeedList();
      })
      .catch(error => {
        document.getElementById("messageContainer").innerHTML =
          '<div class="message">Error reloading container.</div>';
        console.error("Error reloading container:", error);
        btn.disabled = false;
        btn.textContent = originalText;
      });
  });
  
  // Add feed form
  document.getElementById("addFeedForm").addEventListener("submit", function(event) {
    event.preventDefault();
    var btn = document.getElementById("addFeedBtn");
    btn.disabled = true;
    var originalText = btn.textContent;
    btn.textContent = "Adding Feed…";
  
    var youtubeUrlField = document.getElementById("youtubeUrl");
    var updatePeriodField = document.getElementById("update_period");
    var formatField = document.getElementById("format");
    var maxAgeField = document.getElementById("max_age");
    var cleanKeepLastField = document.getElementById("clean_keep_last");
  
    var formData = new URLSearchParams();
    formData.append("youtubeUrl", youtubeUrlField.value);
    if (updatePeriodField.value) formData.append("update_period", updatePeriodField.value);
    if (formatField.value) formData.append("format", formatField.value);
    if (maxAgeField.value) formData.append("max_age", maxAgeField.value);
    if (cleanKeepLastField.value) formData.append("clean_keep_last", cleanKeepLastField.value);
  
    fetch("/add", {
      method: "POST",
      headers: {
        "X-Requested-With": "XMLHttpRequest",
        "Content-Type": "application/x-www-form-urlencoded"
      },
      body: formData.toString()
    })
      .then(response => response.json())
      .then(data => {
        document.getElementById("messageContainer").innerHTML =
          '<div class="message">' + data.message + '</div>';
        btn.disabled = false;
        btn.textContent = originalText;
        youtubeUrlField.value = "";
        updatePeriodField.value = "";
        formatField.selectedIndex = 0;
        maxAgeField.value = "";
        cleanKeepLastField.value = "";
        var adv = document.getElementById("advancedOptions");
        adv.style.display = "none";
        document.getElementById("toggleAdvanced").textContent = "Advanced Options";
        refreshFeedList();
      })
      .catch(error => {
        console.error("Error adding feed:", error);
        document.getElementById("messageContainer").innerHTML =
          '<div class="message">Error adding feed.</div>';
        btn.disabled = false;
        btn.textContent = originalText;
      });
  });
  
  // Remove feed
  function removeFeed(button, feedKey) {
    if (button.innerHTML.trim() !== 'Confirm Remove' && button.innerHTML.trim() !== 'Remove Feed') {
      var originalText = button.innerHTML;
      button.innerHTML = 'Confirm Remove';
      setTimeout(function() {
        if (button.innerHTML.trim() === 'Confirm Remove') {
          button.innerHTML = originalText;
        }
      }, 3000);
      return;
    }
    fetch("/remove", {
      method: "POST",
      headers: {
        "X-Requested-With": "XMLHttpRequest",
        "Content-Type": "application/x-www-form-urlencoded"
      },
      body: "feedKey=" + encodeURIComponent(feedKey)
    })
      .then(response => response.json())
      .then(data => {
        document.getElementById("messageContainer").innerHTML =
          '<div class="message">' + data.message + '</div>';
        refreshFeedList();
      })
      .catch(error => {
        console.error("Error removing feed:", error);
        document.getElementById("messageContainer").innerHTML =
          '<div class="message">Error removing feed.</div>';
      });
  }
  
  // Toggle the visibility of the edit form for a feed item
  function toggleEdit(btnElem, feedKey) {
    var editForm = document.getElementById("edit-form-" + feedKey);
    if (editForm.style.display === "none" || editForm.style.display === "") {
      editForm.style.display = "block";
      btnElem.textContent = "Cancel Edit";
    } else {
      editForm.style.display = "none";
      btnElem.textContent = "Edit Feed";
    }
  }
  
  // Confirm edit feed
  function confirmEdit(feedKey) {
    const prefix = feedKey + "-";
    const updatePeriod   = document.getElementById(prefix + "update_period").value;
    const format         = document.getElementById(prefix + "format").value;
    const maxAge         = document.getElementById(prefix + "max_age").value;
    const cleanKeepLast  = document.getElementById(prefix + "clean_keep_last").value;
  
    var formData = new URLSearchParams();
    formData.append("feedKey", feedKey);
    if (updatePeriod) formData.append("update_period", updatePeriod);
    if (format) formData.append("format", format);
    if (maxAge) formData.append("max_age", maxAge);
    if (cleanKeepLast) formData.append("clean_keep_last", cleanKeepLast);
  
    fetch("/modify", {
      method: "POST",
      headers: {
        "X-Requested-With": "XMLHttpRequest",
        "Content-Type": "application/x-www-form-urlencoded"
      },
      body: formData.toString()
    })
      .then(response => response.json())
      .then(data => {
        document.getElementById("messageContainer").innerHTML =
          '<div class="message">' + data.message + '</div>';
        const editForm = document.getElementById("edit-form-" + feedKey);
        editForm.style.display = "none";
        const editButton = document.querySelector(`[data-role="edit-button"][data-feedkey="${feedKey}"]`);
        if (editButton) editButton.textContent = "Edit Feed";
  
        refreshFeedList();
      })
      .catch(error => {
        console.error("Error modifying feed:", error);
        document.getElementById("messageContainer").innerHTML =
          '<div class="message">Error modifying feed.</div>';
      });
  }