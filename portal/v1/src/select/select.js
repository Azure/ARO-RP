const select_input = $('#enterResourceId');
const select_element = $('#selResourceId');

select_element.css('top', select_input.outerHeight());

select_input.on('keyup change', function () {
    const search_val = $(this).val().toLowerCase();

    if (search_val.length >= 2) {
        select_element.children().each(function () {
            if (!$(this).text().toLowerCase().match(search_val)) {
                $(this).hide();
            } else {
                $(this).show();
            }
        });
    } else {
        select_element.children().each(function () {
            $(this).show();
            select_element.attr('size', select_element.children().length);
        });
    }
});

select_input.focus(function () {
    select_element.attr('size', select_element.children().length);
    select_element.css('z-index', '3');
    select_element.css('visibility', 'visible');

    function reset() {
        select_input.val(select_element.find(":selected").text())
        select_element.css('visibility', 'hidden');
    }

    select_element.change(function () {
        reset();
    });

    select_input.blur(function () {
        setTimeout(function () {
            if (!select_element.is(':focus')) {
                reset();
            }
        }, 50);
    });
});
