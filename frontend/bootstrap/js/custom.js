$(function () {
    $.typeahead({
        input: '.js-typeahead-demo',
        minLength: 1,
        maxItem: 20,
        order: "asc",
        dynamic: true,
        callback: {
            onInit: function (node) {
                console.log('Typeahead Initiated on ' + node.selector);
            }
        },
        source: {
            search: {
                ajax: {
                    url: "/autocomplete",
                    data: {
                        q: "{{query}}"
                    },
                    path: "Matches"
                }
            }
        }
    });
});