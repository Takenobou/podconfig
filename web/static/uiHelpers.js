// Utility functions for UI interactions
export function showMessage(text) {
  const container = document.getElementById('messageContainer');
  container.innerHTML = `<div class="message">${text}</div>`;
}

export function copyText(el, text) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text)
      .then(() => {
        const original = el.innerHTML;
        el.innerHTML = 'Copied XML path to clipboard!';
        setTimeout(() => el.innerHTML = original, 1000);
      })
      .catch(err => console.error('Copy failed:', err));
  } else {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    document.body.appendChild(textarea);
    textarea.select();
    try {
      document.execCommand('copy');
      const original = el.innerHTML;
      el.innerHTML = 'Copied XML path to clipboard!';
      setTimeout(() => el.innerHTML = original, 1000);
    } catch (err) {
      console.error('Fallback copy failed:', err);
    }
    document.body.removeChild(textarea);
  }
}

export function toggleElementDisplay(el, showText, hideText) {
  if (el.style.display === 'none' || el.style.display === '') {
    el.style.display = 'block';
    return hideText;
  } else {
    el.style.display = 'none';
    return showText;
  }
}