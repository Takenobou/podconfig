// Toggle the visibility of advanced options
document.getElementById("toggleAdvanced").addEventListener("click", function(e) {
    e.preventDefault();
    var adv = document.getElementById("advancedOptions");
    if (adv.style.display === "none") {
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
        var successful = document.execCommand("copy");
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
  
  // Refresh the feed list from the /feeds endpoint.
  function refreshFeedList() {
    fetch("/feeds", {
      method: "GET",
      headers: { "X-Requested-With": "XMLHttpRequest" }
    })
    .then(response => response.json())
    .then(feeds => {
      var container = document.getElementById("feedListContainer");
      var html = "<h3>Feeds</h3>";
      if (feeds.length === 0) {
        html += '<div class="message">No feeds found.</div>';
      } else {
        feeds.forEach(function(feed) {
          html += `
<div class="feed-item">
<div class="feed-info">
  <a href="${feed.URL}" target="_blank" class="feed-name">${feed.Name}</a>
  <span class="feed-xml" onclick="copyText(this, '${feed.XMLURL}')">Copy XML path to clipboard</span>
</div>
<button type="button" class="btn-remove" onclick="removeFeed(this, '${feed.Key}')">Remove Feed</button>
</div>
          `;
        });
      }
      container.innerHTML = html;
    })
    .catch(error => {
      console.error("Error refreshing feed list:", error);
      document.getElementById("feedListContainer").innerHTML =
        '<div class="message">Error loading feed list.</div>';
    });
  }
  
  // Refresh feed list on page load.
  refreshFeedList();
  
  // Handle reload button via AJAX.
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
  
  // Handle add feed form via AJAX.
  document.getElementById("addFeedForm").addEventListener("submit", function(event) {
    event.preventDefault();
    var btn = document.getElementById("addFeedBtn");
    btn.disabled = true;
    var originalText = btn.textContent;
    btn.textContent = "Adding Feed…";
    var youtubeUrl = document.getElementById("youtubeUrl").value;
    // Include additional configuration values
    var updatePeriod = document.getElementById("update_period").value;
    var format = document.getElementById("format").value;
    var maxAge = document.getElementById("max_age").value;
    var cleanKeepLast = document.getElementById("clean_keep_last").value;
    var formData = new URLSearchParams();
    formData.append("youtubeUrl", youtubeUrl);
    if(updatePeriod) formData.append("update_period", updatePeriod);
    if(format) formData.append("format", format);
    if(maxAge) formData.append("max_age", maxAge);
    if(cleanKeepLast) formData.append("clean_keep_last", cleanKeepLast);
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
      document.getElementById("youtubeUrl").value = "";
      refreshFeedList();
    })
    .catch(error => {
      document.getElementById("messageContainer").innerHTML =
        '<div class="message">Error adding feed.</div>';
      console.error("Error adding feed:", error);
      btn.disabled = false;
      btn.textContent = originalText;
    });
  });
  
  function removeFeed(button, feedKey) {
    if (button.innerHTML.trim() !== 'Confirm Remove' && button.innerHTML.trim() !== 'Remove Feed') {
    }
    if (button.innerHTML.trim() !== 'Confirm Remove') {
      var originalText = button.innerHTML;
      button.innerHTML = 'Confirm Remove';
      setTimeout(function() {
        if (button.innerHTML.trim() === 'Confirm Remove') {
          button.innerHTML = originalText;
        }
      }, 3000);
      return;
    }
    fetch('/remove', {
      method: 'POST',
      headers: {
        'X-Requested-With': 'XMLHttpRequest',
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      body: 'feedKey=' + encodeURIComponent(feedKey)
    })
    .then(response => response.json())
    .then(data => {
      document.getElementById('messageContainer').innerHTML =
        '<div class="message">' + data.message + '</div>';
      refreshFeedList();
    })
    .catch(error => {
      console.error('Error removing feed:', error);
      document.getElementById('messageContainer').innerHTML =
        '<div class="message">Error removing feed.</div>';
    });
  }