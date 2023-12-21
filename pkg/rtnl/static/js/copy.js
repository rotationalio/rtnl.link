function copyUrlToClipboard() {
    const shortUrl = document.getElementById('short-url').innerHTML;
    navigator.clipboard.writeText(shortUrl);

    const shortIcon = document.getElementById('short-icon');
    shortIcon.classList.remove('fa-copy');
    shortIcon.classList.add('fa-circle-check');
    setTimeout(() => {
        shortIcon.classList.remove('fa-circle-check');
        shortIcon.classList.add('fa-copy');
    }, 1000);
}

function copyAltUrlToClipboard() {
    const altShortUrl = document.getElementById('alt-short-url').innerHTML;
    navigator.clipboard.writeText(altShortUrl);

    const altShortIcon = document.getElementById('alt-short-icon');
    altShortIcon.classList.remove('fa-copy');
    altShortIcon.classList.add('fa-circle-check');
    setTimeout(() => {
        altShortIcon.classList.remove('fa-circle-check');
        altShortIcon.classList.add('fa-copy');
    }, 1000);
}