function InitDialogs() {
    var confirmHarvestDialog, plantInfoDialog;
    confirmHarvestDialog = $("#confirm-harvest").dialog({
        autoOpen: false,
        model: true,
        show: {
            effect: "drop",
            duration: 500
        },
        hide: {
            effect: "explode",
            duration: 1000
        },
        buttons: {
            "Yes": function() {
		$( this ).dialog( "close" );
            },
            "No": function() {
		$( this ).dialog( "option", "hide", {effect: "drop", duration: 500});
		$( this ).dialog( "close" );
            }
        }
    });
    
    plantInfoDialog = $("#plant-info").dialog({
        autoOpen: false,
        modal: true,
        show: {
            effect: "drop",
            duration: 500
        },
        hide: {
            effect: "drop",
            duration: 1000
        },
        buttons: {
            "Harvest": function() {
		confirmHarvestDialog.find("#confirmHarvest-Name").text($( this).find("#plantInfo-Name").text());
		confirmHarvestDialog.dialog("open");
		$( this ).dialog( "close" );
            },
            "OK": function() {
		$( this ).dialog( "close" );
            }
        }
    });
    $("button.slot-plant").on("click", function( event ) {
        event.preventDefault();
        plantInfoDialog.find("#plantInfo-Name").text(event.currentTarget.dataset.name);
        plantInfoDialog.find("#plantInfo-Planted").text($.datepicker.formatDate('dd M yy', new Date(event.currentTarget.dataset.plantingtime*1000)));
        plantInfoDialog.find("#plantInfo-HarvestFrom").text($.datepicker.formatDate('dd M yy', new Date(event.currentTarget.dataset.harvestfrom*1000)));
        plantInfoDialog.find("#plantInfo-HarvestBy").text($.datepicker.formatDate('dd M yy', new Date(event.currentTarget.dataset.harvestby*1000)));
        plantInfoDialog.dialog("open");
    });
}

function plantUpdate(e) {
    var data = jQuery.parseJSON(e.data);
}

function StartStream() {
    if (!window.EventSource) {
	alert("EventSource is not enabled in this browser");
	return;
    }
    var stream = new EventSource('/stream/');
    stream.addEventListener('plant', plantUpdate, false);
}
