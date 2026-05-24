document.querySelectorAll('details.embed-collapsible').forEach(function(el) {
  el.addEventListener('toggle', function() {
    if (el.open) {
      el.querySelectorAll('iframe[data-src]').forEach(function(iframe) {
        iframe.src = iframe.dataset.src;
        iframe.removeAttribute('data-src');
      });
    }
  });
});
