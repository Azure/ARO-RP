import 'bootstrap/dist/css/bootstrap.min.css';

import 'bootstrap/js/dist/util';
import 'bootstrap/js/dist/dropdown';
import 'bootstrap/js/dist/alert';

import "./select/select.css"
import 'tom-select/dist/css/tom-select.bootstrap5.min.css';
import TomSelect from 'tom-select/dist/js/tom-select.complete.min';

jQuery.extend({
    redirect: function (location, args) {
        var form = $("<form method='POST' style='display: none;'></form>");
        form.attr("action", location);

        $.each(args || {}, function (key, value) {
            var input = $("<input name='hidden'></input>");

            input.attr("name", key);
            input.attr("value", value);

            form.append(input);
        });

        form.append($("input[name='gorilla.csrf.Token']").first());
        form.appendTo("body").submit();
    }
});

jQuery(function () {
    $.ajax({
        url: "/api/clusters",
        success: function (clusters) {
            $.each(clusters, function (i, cluster) {
                $("#selResourceId").append($("<option>").text(cluster.resourceId));
            });

            $("#selResourceId").prop("disabled", false);

            new TomSelect("#selResourceId", {
                plugins: ["dropdown_input"],
                maxOptions: null,
                maxItems: 1,
                placeholder: "Search clusters...",
                onDropdownOpen: function (value) {
                    $(".dropdown-input").val($(".ts-control").children("div").html());
                }
            });

            $("#selResourceId").css("display", "none");
        },
        dataType: "json",
    });

    $("#btnLogout").click(function () {
        $.redirect("/api/logout");
    });

    $("#btnKubeconfig").click(function () {
        $.redirect($("#selResourceId").val() + "/kubeconfig/new");
    });

    $("#btnPrometheus").click(function () {
        window.location = $("#selResourceId").val() + "/prometheus";
    });

    $("#btnSSH").click(function () {
        $.ajax({
            method: "POST",
            url: $("#selResourceId").val() + "/ssh/new",
            headers: {
                "X-CSRF-Token": $("input[name='gorilla.csrf.Token']").val(),
            },
            contentType: "application/json",
            data: JSON.stringify({
                "master": parseInt($("#selMaster").val()),
            }),
            success: function (reply) {
                if (reply["error"]) {
                    var template = $("#tmplSSHAlertError").html();
                    var alert = $(template);

                    alert.find("span[data-copy='error']").text(reply["error"]);
                    $("#divAlerts").html(alert);

                    return;
                }

                var template = $("#tmplSSHAlert").html();
                var alert = $(template);

                alert.find("span[data-copy='command'] > code").text(reply["command"]);
                alert.find("span[data-copy='command']").attr("data-copy", reply["command"]);
                alert.find("span[data-copy='password'] > code").text("********");
                alert.find("span[data-copy='password']").attr("data-copy", reply["password"]);
                $("#divAlerts").html(alert);

                $('.copy-button').click(function () {
                    var textarea = $("<textarea class='style: hidden;' id='textarea'></textarea>");
                    textarea.text($(this).next().attr("data-copy"));
                    textarea.appendTo("body");

                    textarea = document.getElementById("textarea")
                    textarea.select();
                    textarea.setSelectionRange(0, textarea.value.length + 1);
                    document.execCommand('copy');
                    document.body.removeChild(textarea)
                });
            },
            dataType: "json",
        });
    });

    $("#btnV2").click(function () {
        window.location = "/";
    });
});
