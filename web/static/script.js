import { fetchFeeds, addFeed, modifyFeed, removeFeedAPI, reloadContainer, fetchChangelog } from './feedApi.js';
import { showMessage, copyText, toggleElementDisplay } from './uiHelpers.js';

// Toggle the visibility of advanced options in the add form
const toggleLink = document.getElementById("toggleAdvanced");
toggleLink.addEventListener("click", e => {
  e.preventDefault();
  const adv = document.getElementById("advancedOptions");
  toggleLink.textContent = toggleElementDisplay(adv, "Advanced Options", "Hide Advanced Options");
});

function attachFeedListEventListeners() {
  // Edit, save, remove, copy XML handlers
  document.querySelectorAll('[data-role="edit-button"]').forEach(btn => {
    btn.addEventListener("click", () => {
      const key = btn.dataset.feedkey;
      const form = document.getElementById(`edit-form-${key}`);
      btn.textContent = toggleElementDisplay(form, "Edit Feed", "Cancel Edit");
      setupEditFormChangeListeners(key);
    });
  });
  document.querySelectorAll('[data-role="save-edit"]').forEach(btn => {
    btn.addEventListener("click", () => confirmEdit(btn.dataset.feedkey));
  });
  document.querySelectorAll('[data-role="remove-feed"]').forEach(btn => {
    btn.addEventListener("click", () => removeFeed(btn));
  });
  document.querySelectorAll('[data-role="xml-button"]').forEach(el => {
    el.addEventListener("click", () => copyText(el, el.dataset.xmlurl));
  });
}

async function refreshFeedList() {
  try {
    const html = await fetchFeeds();
    document.getElementById("feedListWrapper").innerHTML = html;
    attachFeedListEventListeners();
  } catch (err) {
    console.error(err);
    document.getElementById("feedListWrapper").innerHTML = '<div class="message">Error loading feed list.</div>';
  }
}

async function refreshChangelogWrapper() {
  try {
    const html = await fetchChangelog();
    document.getElementById("changelogWrapper").innerHTML = html;
  } catch (err) {
    console.error(err);
    document.getElementById("changelogWrapper").innerHTML = '';
  }
}

// Initial load
refreshFeedList();
refreshChangelogWrapper();

// Reload container
const reloadBtn = document.getElementById("reloadBtn");
reloadBtn.addEventListener("click", async () => {
  reloadBtn.disabled = true;
  const txt = reloadBtn.textContent;
  reloadBtn.textContent = "Reloading Container…";
  try {
    const data = await reloadContainer();
    showMessage(data.message);
    await refreshFeedList();
    await refreshChangelogWrapper();
  } catch (err) {
    console.error(err);
    showMessage('Error reloading container.');
  } finally {
    reloadBtn.disabled = false;
    reloadBtn.textContent = txt;
  }
});

// Add feed form
const addForm = document.getElementById("addFeedForm");
addForm.addEventListener("submit", async e => {
  e.preventDefault();
  const btn = document.getElementById("addFeedBtn");
  btn.disabled = true;
  const orig = btn.textContent;
  btn.textContent = "Adding Feed…";
  const params = {
    youtubeUrl: document.getElementById("youtubeUrl").value,
    update_period: document.getElementById("update_period").value,
    format: document.getElementById("format").value,
    max_age: document.getElementById("max_age").value,
    clean_keep_last: document.getElementById("clean_keep_last").value
  };
  try {
    const data = await addFeed(params);
    showMessage(data.message);
    addForm.reset();
    toggleElementDisplay(document.getElementById("advancedOptions"), "Advanced Options", "Hide Advanced Options");
    await refreshFeedList();
    await refreshChangelogWrapper();
  } catch (err) {
    console.error(err);
    showMessage('Error adding feed.');
  } finally {
    btn.disabled = false;
    btn.textContent = orig;
  }
});

function removeFeed(btn) {
  if (btn.textContent.trim() !== 'Confirm Remove') {
    const orig = btn.textContent;
    btn.textContent = 'Confirm Remove';
    setTimeout(() => btn.textContent === 'Confirm Remove' && (btn.textContent = orig), 3000);
    return;
  }
  (async () => {
    try {
      const data = await removeFeedAPI(btn.dataset.feedkey);
      showMessage(data.message);
      await refreshFeedList();
      await refreshChangelogWrapper();
    } catch (err) {
      console.error(err);
      showMessage('Error removing feed.');
    }
  })();
}

function setupEditFormChangeListeners(key) {
  const prefix = `${key}-`;
  const fields = ['update_period','format','max_age','clean_keep_last'];
  const btn = document.querySelector(`[data-role="save-edit"][data-feedkey="${key}"]`);
  const check = () => {
    btn.disabled = !fields.some(f => document.getElementById(prefix+f).value !== document.getElementById(prefix+f).dataset.original);
  };
  fields.forEach(f => document.getElementById(prefix+f)?.addEventListener(f==='format'?'change':'input', check));
  check();
}

function confirmEdit(key) {
  const prefix = `${key}-`;
  const params = { feedKey: key };
  ['update_period','format','max_age','clean_keep_last'].forEach(f => {
    const v = document.getElementById(prefix+f).value;
    if (v) params[f] = v;
  });
  (async () => {
    try {
      const data = await modifyFeed(params);
      showMessage(data.message);
      document.getElementById(`edit-form-${key}`).style.display = 'none';
      document.querySelector(`[data-role="edit-button"][data-feedkey="${key}"]`).textContent = 'Edit Feed';
      await refreshFeedList();
      await refreshChangelogWrapper();
    } catch (err) {
      console.error(err);
      showMessage('Error modifying feed.');
    }
  })();
}