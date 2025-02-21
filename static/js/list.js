const items = document.querySelectorAll('.list-item');

// aggregate tags from items
let tagMap = new Map()

items.forEach(item => {
    const tags = item.querySelectorAll('.item-tag');
    tags.forEach(tag => {
        if (tagMap.has(tag.innerHTML)) {
            tagMap.get(tag.innerHTML).push(item);
        } else {
            tagMap.set(tag.innerHTML, [item]);
        }
    });
});

// get the div that holds the tags
const tagsDiv = document.getElementById('tags');

// create a select all checkbox
const selectAllCheckbox = appendCheckbox(tagsDiv, 'all', 'select-all', 'select-all');
selectAllCheckbox.checked = true;

selectAllCheckbox.addEventListener('change', () => {
    const checkboxes = tagsDiv.querySelectorAll('.tag-checkbox');
    checkboxes.forEach(checkbox => {
        checkbox.checked = selectAllCheckbox.checked;
    });
    renderItems();
});

// sort tags
const sortedTags = Array.from(tagMap.keys()).sort();

// create a checkbox for each tag
sortedTags.forEach(tag => {
    const checkbox = appendCheckbox(tagsDiv, tag, tag, "tag-checkbox");
    checkbox.checked = true;
    // re-render items when a tag is checked/unchecked
    checkbox.addEventListener('change', renderItems);
});

// get the div that holds the search input
const searchDiv = document.getElementById('search');

// create a search text input
const searchInput = document.createElement('input');
searchInput.type = 'text';
searchInput.placeholder = 'Search';
searchInput.id = 'search-input';
searchDiv.appendChild(searchInput);

// re-render items based on searchInput
searchInput.addEventListener('input', renderItems);

function appendCheckbox(parent, label, value, cssClass) {
    const labelElement = document.createElement('label');
    const checkbox = document.createElement('input');
    checkbox.type = 'checkbox';
    checkbox.value = value;
    if (cssClass) {
        checkbox.classList.add(cssClass);
    }
    labelElement.appendChild(document.createTextNode(label));
    labelElement.appendChild(checkbox);
    parent.appendChild(labelElement);

    return checkbox;
}

function getSelectedTags() {
    return Array.from(document.querySelectorAll('.tag-checkbox:checked')).map(checkbox => checkbox.value);
}

function getSearchTerm() {
    return searchInput.value.toLowerCase().trim();
}

function renderItems() {
    const searchTerm = getSearchTerm();
    const selectedTags = getSelectedTags();

    let numItems = 0;
    items.forEach(item => {
        const tags = item.querySelectorAll('.item-tag');
        const titleMatchesSearchTerm = matchesSearchTerm(item.querySelector('.item-name').innerText, searchTerm);
        const itemDescriptionElement = item.querySelector('.item-description');
        const descriptionMatchesSearchTerm = itemDescriptionElement && matchesSearchTerm(itemDescriptionElement.innerText, searchTerm);
        const tagMatchesSearchTerm = matchesSearchTerm(item.querySelector('.item-tags').innerText, searchTerm);
        const tagIsSelected = Array.from(tags).some(tag => selectedTags.includes(tag.innerText));
        const show = tagIsSelected && (titleMatchesSearchTerm || descriptionMatchesSearchTerm || tagMatchesSearchTerm);
        item.style.display = show ? 'block' : 'none';
        if (show) {
            numItems++;
        }
    })

    if (numItems === 0) {
        showNoResults(`No ${listType} match your filters.`);
    } else {
        hideNoResults();
    }
}

function matchesSearchTerm(s, searchTerm) {
    if(!searchTerm) return true;
    return s.toLowerCase().includes(searchTerm);
}

function showNoResults(message) {
    const noItemsElement = document.querySelector('#no-items');
    const p = noItemsElement.querySelector('p');
    p.innerText = message;
    noItemsElement.style.display = 'block';
}

function hideNoResults() {
    const noItemsElement = document.querySelector('#no-items');
    noItemsElement.style.display = 'none';
}