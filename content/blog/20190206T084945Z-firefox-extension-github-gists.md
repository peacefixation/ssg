---
title: A Github Gists extension for Firefox
date: 2019-02-06T19:49:45+11:00
tags:
  - firefox
  - github
description: Creating a Firefox extension to list and search your Github Gists.
draft: false
---

I store snippets of code with Github Gists to remind myself how to do things that I don't do regularly enough to remember off hand. At first it was only a few, but my list is growing and as it grows it becomes harder to find the one I want. I've made myself a little Firefox extension to list all my Gists with a search field so I can quickly locate them.

Making a Firefox extension is easier than ever since they've adopted the same API as Chrome. There's no more XUL; it's all Javascript, HTML and CSS and you can get something up and running in no time.

My spec was short and sweet:
- get the user's Github repo
- search Github for the Gists
- provide a browser button with a popup that shows the Gists in a list
- filter the list with a search field
- open a Gist in a new window when clicked

Inspection of the Github API reveals two distinct options. The v3 REST API and a new v4 GraphQL API that doesn't have as many features, at least as far as Gists are concerned. You can retrieve metadata but not the contents of a Gist and there's no mutation features (create, edit, etc). I'm not writing a Gist editor and I was curious about GraphQL so I decided to use the GraphQL API for this extension.

With my spec defined and a broad overview in mind, it's time to create a basic extension.

## Manifest

An extension requires a `manifest.json` file to describe itself to the browser. Here's the manifest for my Firefox extension with some comments to describe each section.

```json
{
    "manifest_version": 2, // always 2
    "name": "Github Gists", // the extension name
    "version": "0.4", // the extension version (important for updates to work correctly)

    "description": "Search your Github Gists",

    // various icons for use by Firefox
    "icons": {
        "32": "icons/GitHub-Mark-32px.png",
        "64": "icons/GitHub-Mark-64px.png",
        "120": "icons/GitHub-Mark-120px-plus.png"
    },

    // the background page, launches when the addon launches and persists while the browser is open and the extension is enabled
    "background": {
        "scripts": [
            "js/background.js"
        ]
    },

    // the options page for the extension in about:addons
    "options_ui": {
        "page": "html/options.html"
    },

    // a button that opens a popup
    "browser_action": {
		"default_title": "Gists",
		"default_icon": "icons/GitHub-Mark-64px.png", // the button icon
		"default_popup": "html/menu.html" // the popup page
	},

    // a unique id to identify the extension
    "applications": {
        "gecko": {
          "id": "gists@peacefixation"
        }
      }
}
```


## Background Page

The background "page" is a JavaScript file that runs in the background and is the engine of the extension that does the work of sending web requests and keeps track of state. The other pages interact with the background page with `chrome.extension.getBackgroundPage()` when they need to initiate a web request to retrieve new data, or access state.

My background page has a method to download Gists and store them in memory, and an accessor method to get the Gists that were downloaded. The download method executes an AJAX request to the Github API with a GraphQL query. The GraphQL query will return a maximum of 100 Gists. To retrieve more, you must request a cursor that indicates the end of the page that you retrieved, and a boolean variable that indicates if any more pages are available, then send a new request with a parameter called `after` that is populated with the cursor. In this way you can retrieve pages of 100 Gists at a time until there are no more pages. I decided to limit the number of requests that I send to 10 so that I don't spam the API too hard, so my extension will display a maximum of 1000 Gists which I hope is reasonable.

Github's API documentation includes a section on [writing GraphQL queries](https://developer.github.com/v4/guides/forming-calls/) and a section on the format of the [Gist object](https://developer.github.com/v4/object/gist/) that you can request. I crafted my own query with variables to retrieve the first `x` Gists after the given cursor.

```javascript
query ($first: Int, $after: String) { viewer { gists(first:$first, after:$after, privacy:ALL) { edges { node { id description name pushedAt owner { resourcePath } } } pageInfo { endCursor hasNextPage } } } }
```

To retrieve the Gists I craft an AJAX request to the Github API and include the Authorization header with a personal access token. Each user must create a personal access token from their own Github account to give the extension permission to access it on their behalf. I send the request, and if there are more pages available, I send another request recursively until no more pages are available or I reach my maxRequests limit.

```javascript
function requestGists(first, after) {
    let xhttp = new XMLHttpRequest();
    
    xhttp.onreadystatechange = function() {
        if(this.readyState == 4) {
            
            localStorage["requestStatus"] = this.status;

            if(this.status == 200) {
                let json = JSON.parse(this.responseText);

                // if the errors field is populated stop requesting more gists
                if(json["errors"]) {
                    localStorage["requestStatus"] = json["errors"][0]["message"];
                    browser.runtime.sendMessage({"action": "checkStatus"});
                    return;
                }

                let hasNextPage = json["data"]["viewer"]["gists"]["pageInfo"]["hasNextPage"];
                let endCursor = json["data"]["viewer"]["gists"]["pageInfo"]["endCursor"];

                // append the new gists
                for(let i = 0; i < json["data"]["viewer"]["gists"]["edges"].length; i++) {
                    gists.push(json["data"]["viewer"]["gists"]["edges"][i]);
                }

                // keep requesting gists while there are more pages
                if(hasNextPage === true && requestNum < maxRequests) {
                    requestNum++;
                    requestGists(100, endCursor);
                }
            }

            browser.runtime.sendMessage({"action": "checkStatus"});
        }
    };

    xhttp.open("POST", "https://api.github.com/graphql", true);
    xhttp.setRequestHeader("Authorization", "Bearer " + localStorage["token"]);
    xhttp.setRequestHeader("Content-Type", "application/json");
 
     // the GraphQL query
    let query = "query ($first: Int, $after: String) { viewer { gists(first:$first, after:$after, privacy:ALL) { edges { node { id description name pushedAt owner { resourcePath } } } pageInfo { endCursor hasNextPage } } } }"
    let request = JSON.stringify({
        query: query,
        variables: { first: first, after: after }
    });

    xhttp.send(request);
}
```

## Menu Page

The menu page is shown when the browser button is pressed. It's a simple HTML page with some JavaScript to control it (add/remove DOM elements, event handler for search function).

## Options Page

The options page is shown on the Firefox `about:addons` page and is another simple HTML page with accompanying JavaScript. You can open the options page programatically by calling `browser.runtime.openOptionsPage()`. In my menu page I show a message if the user's Github token is invalid and provide a link that opens the Options page.

## Development and debugging

You can load your extension temporarily on the `about:debugging` page. Press the "Load Temporary Addon-on..." button and select your `manifest.json` file from the file browser. Once loaded you can open a JavaScript console by pressing the "Debug" button. If you make changes to your code you can reload the extension by pressing the "Reload" button.

If you're interested in learning more see the [Firefox developer documentation](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions). You can find the source code for my extension on [Github](https://github.com/peacefixation/firefox-github-gists). You can install the extension from the [Firefox addons page](https://addons.mozilla.org/en-US/firefox/addon/github-gists/).
