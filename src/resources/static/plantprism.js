var plantDB;

function processPlantDB(data) {
    plantDB = data;
    var ptSel = $("#plantType");
    $.each(plantDB, function(i, plant) {
	var o = new Option(plant.Names["de"], i);
	ptSel.append($(o));
    });
}

function FetchPlantDB() {
    $.getJSON("plantdb.json", processPlantDB);
}

function InitDialogs() {
    var addPlantDialog, confirmHarvestDialog, plantInfoDialog;
    addPlantDialog = $("#add-plant").dialog({
	autoOpen: false,
	modal: true,
	show: {
	    effect: "drop",
	    duration: 500
	},
	buttons: {
	    "OK": function() {
		$.post("addPlant", $( this ).find("form").serialize());
		$( this ).dialog( "option", "hide", {effect: "scale", duration: 1000});
		$( this ).dialog( "close" );
	    },
	    "Cancel": function() {
		$( this ).dialog( "option", "hide", {effect: "drop", duration: 500});
		$( this ).dialog( "close" );
	    }
	}
    });

    confirmHarvestDialog = $("#confirm-harvest").dialog({
        autoOpen: false,
        modal: true,
        show: {
            effect: "drop",
            duration: 500
        },
        buttons: {
            "Yes": function() {
		$( this ).dialog( "option", "hide", {effect: "explode", duration: 1000});
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
    $("button.slot-empty").on("click", function( event ) {
	event.preventDefault();
	addPlantDialog.find("#slot").val(event.currentTarget.dataset.slot);
	addPlantDialog.dialog("open");
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
