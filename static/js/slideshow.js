document.querySelectorAll('[data-slideshow]').forEach(function(el) {
  var slides = el.querySelectorAll('.slide');
  if (slides.length === 0) return;

  var current = 0;
  var counter = el.querySelector('.slideshow-counter');

  // Returns the gap (px) between the bottom of the object-fit:contain image
  // and the bottom of its container. Zero for images that fill the full height.
  function imgBottomGap(img, slide) {
    if (!img.naturalWidth || !img.naturalHeight) return 0;
    var iAR = img.naturalWidth / img.naturalHeight;
    var cAR = slide.offsetWidth / slide.offsetHeight;
    var renderedH = iAR > cAR
      ? slide.offsetWidth / iAR   // landscape: width-constrained
      : slide.offsetHeight;       // portrait: height-constrained
    return Math.max(0, (slide.offsetHeight - renderedH) / 2);
  }

  function positionOverlays(index) {
    var slide = slides[index];
    var img = slide.querySelector('img');
    if (!img) return;
    var bottom = imgBottomGap(img, slide) + 10;
    var caption = slide.querySelector('.slide-caption');
    if (caption) caption.style.bottom = bottom + 'px';
    if (counter) counter.style.bottom = bottom + 'px';
  }

  function show(index) {
    slides[current].classList.remove('slide-active');
    current = (index + slides.length) % slides.length;
    slides[current].classList.add('slide-active');
    if (counter) counter.textContent = (current + 1) + ' / ' + slides.length;
    positionOverlays(current);
  }

  // Re-position after image loads (handles lazy-loaded and cached images).
  slides.forEach(function(slide, i) {
    var img = slide.querySelector('img');
    if (!img) return;
    if (img.complete && img.naturalWidth) {
      if (i === current) positionOverlays(i);
    } else {
      img.addEventListener('load', function() {
        if (i === current) positionOverlays(i);
      });
    }
  });

  window.addEventListener('resize', function() { positionOverlays(current); });

  show(0);

  var prev = el.querySelector('.slideshow-prev');
  var next = el.querySelector('.slideshow-next');
  if (prev) {
    prev.addEventListener('mousedown', function(e) { e.preventDefault(); });
    prev.addEventListener('click', function() { show(current - 1); });
  }
  if (next) {
    next.addEventListener('mousedown', function(e) { e.preventDefault(); });
    next.addEventListener('click', function() { show(current + 1); });
  }

  el.addEventListener('keydown', function(e) {
    if (e.key === 'ArrowLeft')  show(current - 1);
    if (e.key === 'ArrowRight') show(current + 1);
  });
  el.setAttribute('tabindex', '0');
});
