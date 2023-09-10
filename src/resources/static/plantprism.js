var plantDB;
var addPlantDialog, confirmHarvestDialog, plantInfoDialog;

var plantClick = function( event ) {
        event.preventDefault();
        plantInfoDialog.find("#slot").val(event.currentTarget.dataset.slot);
        plantInfoDialog.find("#plantInfo-Name").text(event.currentTarget.dataset.name);
        plantInfoDialog.find("#plantInfo-Planted").text($.datepicker.formatDate('dd M yy', new Date(event.currentTarget.dataset.plantingtime*1000)));
        plantInfoDialog.find("#plantInfo-HarvestFrom").text($.datepicker.formatDate('dd M yy', new Date(event.currentTarget.dataset.harvestfrom*1000)));
        plantInfoDialog.find("#plantInfo-HarvestBy").text($.datepicker.formatDate('dd M yy', new Date(event.currentTarget.dataset.harvestby*1000)));
        plantInfoDialog.dialog("open");
};

var emptyClick = function( event ) {
	event.preventDefault();
	addPlantDialog.find("#id").val(deviceID);
	addPlantDialog.find("#slot").val(event.currentTarget.dataset.slot);
	addPlantDialog.dialog("open");
};

function processPlantDB(data) {
    plantDB = data;
    var ptSel = $("#plantType");
    $.each(plantDB, function(id, plant) {
	var o = new Option(plant.Names["de"], id);
	ptSel.append($(o));
    });
}

function FetchPlantDB() {
    $.getJSON("plantdb.json", processPlantDB);
}

function InitUI() {
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
		$.post("harvestPlant", $( this ).find("form").serialize());
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
		confirmHarvestDialog.find("#id").val(deviceID);
		confirmHarvestDialog.find("#slot").val($(this).find("#slot").val());
		confirmHarvestDialog.find("#confirmHarvest-Name").text($(this).find("#plantInfo-Name").text());
		confirmHarvestDialog.dialog("open");
		$( this ).dialog( "close" );
            },
            "OK": function() {
		$( this ).dialog( "close" );
            }
        }
    });
    $("button.slot-plant").on("click", plantClick);
    $("button.slot-empty").on("click", emptyClick);
    $("#tabs").tabs();
}

function slotEvent(e) {
    var data = jQuery.parseJSON(e.data);
    var btn = $("#slot-" + data["Slot"]);
    var img = btn.find("img");
    var div = btn.find("div");
    if (data["Planted"]) {
	btn.attr("class", "slot-plant");
	btn.attr("data-name", data["PlantName"]);
	btn.attr("data-plantingtime", data["PlantingTime"]);
	btn.attr("data-harvestfrom", data["HarvestFrom"]);
	btn.attr("data-harvestby", data["HarvestBy"]);
	btn.off("click");
	btn.on("click", plantClick);
	img.attr("src", "static/sprout.png");
	img.attr("class", "sprout");
	div.text(data["PlantName"]);
    } else {
	btn.attr("class", "slot-empty");
	btn.off("click");
	btn.on("click", emptyClick);
	img.attr("src", "static/blank.png");
	img.attr("class", "empty");
	div.html("&nbsp;");
    }
}

function statusEvent(e) {
    var data = jQuery.parseJSON(e.data);
    $("#tempA").text(data["TempA"]);
    $("#tempB").text(data["TempB"]);
    $("#tempTank").text(data["TempTank"]);
    $("#humidA").text(data["HumidA"]);
    $("#humidB").text(data["HumidB"]);
    var tl0 = (data["TankLevel"]>=1) ? "full" : "empty";
    $("#tankLevel0").attr("class", "tankBlock "+tl0);
    var tl1 = (data["TankLevel"]==2) ? "full" : "empty";
    $("#tankLevel1").attr("class", "tankBlock "+tl1);
    if (data["LightA"]) {
	$("#lightA").text("🌞");
    } else {
	$("#lightA").text("🌛");
    }
    if (data["LightB"]) {
	$("#lightB").text("🌞");
    } else {
	$("#lightB").text("🌛");
    }
    $("#ec").text(data["EC"]);
    $("#smoothedEC").text(data["SmoothedEC"].toFixed(1));
    $("#wantNutrient").text(data["WantNutrient"]);
}

function StartStream() {
    if (!window.EventSource) {
	alert("EventSource is not enabled in this browser");
	return;
    }
    var stream = new EventSource('/stream?id='+deviceID);
    stream.addEventListener('slot', slotEvent, false);
    stream.addEventListener('status', statusEvent, false);
}
