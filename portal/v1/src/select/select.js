const select_input = $("#enterResourceId");
const select_element = $("#selResourceId");

select_element.css("top", select_input.outerHeight());

function selectResourceId() {
    select_input.val(select_element.find(":selected").text());
    select_element.css("visibility", "hidden");
}

select_input.on("keyup change", function () {
    if (select_input.val().length >= 2) {
        select_element.children().each(function (i) {
            if (i != 0) {
                if (!$(this).text().toLowerCase().match(select_input.val().toLowerCase())) {
                    $(this).hide();
                } else {
                    $(this).show();
                }
            }
        });
    } else {
        select_element.children().each(function () {
            $(this).show();
            select_element.attr("size", select_element.children().length);
        });
    }
});

select_input.focus(function () {
    select_element.attr("size", select_element.children().length);
    select_element.css("z-index", "3");
    select_element.css("visibility", "visible");
});

select_element.change(function () {
    selectResourceId();
});

select_input.blur(function () {
    setTimeout(function () {
        if (!select_element.is(":focus")) {
            selectResourceId();
        }
    }, 50);
});
