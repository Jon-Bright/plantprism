#include <stdio.h>
#include <stdint.h>
#include <stdbool.h>
#include <time.h>

// Recipe from Sat  1 Jul 18:40:12 CEST 2023
// cycle_start Thu 29 Dec 01:00:00 CET 2022
uint8_t recipe_a[]={
	0xec, 0x56, 0xa0, 0x64, 0x80, 0xd8, 0xac, 0x63,
	0x02, 0x07, 0x02, 0x01, 0x00, 0x01, 0x5e, 0x02,
	0x24, 0x02, 0xba, 0x80, 0x51, 0x01, 0x00, 0x00,
	0x00, 0x00, 0x00, 0xfc, 0x08, 0x3c, 0x00, 0xff,
	0xff, 0xf8, 0xd9, 0x00, 0x00, 0x3d, 0x27, 0x21,
	0x0a, 0xfc, 0x08, 0x46, 0x00, 0x83, 0x70, 0x88,
	0x77, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xd0,
	0x07, 0x00, 0x00, 0x80, 0x70, 0xf8, 0xd9, 0x00,
	0x00, 0x3d, 0x27, 0x21, 0x0a, 0xfc, 0x08, 0x46,
	0x00, 0x83, 0x70, 0x88, 0x77, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0xd0, 0x07, 0x00, 0x00, 0x80,
	0x70};

// Recipe from Sat 22 Jul 19:28:10 CEST 2023
// cycle_start Sun  2 Apr 02:00:00 CEST 2023
uint8_t recipe_b[]={
	0xaa, 0x11, 0xbc, 0x64, 0x80, 0xc5, 0x28, 0x64,
	0x02, 0x07, 0x01, 0x02, 0x00, 0x02, 0x24, 0x01,
	0x54, 0x02, 0x08, 0xf8, 0xd9, 0x00, 0x00, 0x3d,
	0x27, 0x21, 0x0a, 0xfc, 0x08, 0x46, 0x00, 0x83,
	0x70, 0x88, 0x77, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xd0, 0x07, 0x00, 0x00, 0x80, 0x70, 0x80,
	0x51, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0xfc,
	0x08, 0x3c, 0x00, 0xff, 0xff, 0xf8, 0xd9, 0x00,
	0x00, 0x3d, 0x27, 0x21, 0x0a, 0xfc, 0x08, 0x46,
	0x00, 0x83, 0x70, 0x88, 0x77, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0xd0, 0x07, 0x00, 0x00, 0x80,
	0x70};

uint8_t recipe_gen[]={
	0x86, 0x7b, 0xd6, 0x64, 0x80, 0x3f, 0xcc, 0x64,
	0x02, 0x07, 0x02, 0x02, 0x00, 0x01, 0x06, 0x02,
	0x64, 0x01, 0x06, 0x02, 0x64, 0x80, 0x51, 0x01,
	0x00, 0x00, 0x00, 0x00, 0x00, 0xfc, 0x08, 0x46,
	0x00, 0xff, 0xff, 0xf0, 0xd2, 0x00, 0x00, 0x01,
	0x02, 0x03, 0x04, 0xfc, 0x08, 0x46, 0x00, 0x80,
	0x70, 0x90, 0x7e, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xd0, 0x07, 0x00, 0x00, 0x80, 0x70, 0x80,
	0x51, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0xfc,
	0x08, 0x46, 0x00, 0xff, 0xff, 0xf0, 0xd2, 0x00,
	0x00, 0x01, 0x02, 0x03, 0x04, 0xfc, 0x08, 0x46,
	0x00, 0x80, 0x70, 0x90, 0x7e, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0xd0, 0x07, 0x00, 0x00, 0x80,
	0x70};

typedef uint8_t byte;
typedef uint8_t undefined;
typedef uint32_t uint;

enum Layer {
	LAYER_A=0,
	LAYER_B=1,
	APPLIANCE=2,
	LAYER_MAX=3
};

struct recipe_appliance_period {
	uint32_t length_seconds;
};

struct recipe_layer_period {
	uint32_t length_seconds;
	byte light_0;
	byte light_1;
	byte light_2;
	byte light_3;
	uint16_t temp_target;
	uint16_t water_target_a;
	uint16_t water_target_b;
};

struct recipe_layer_fragment {
	struct recipe_layer_period * layer_period;
	struct recipe_appliance_period * appliance_period;
	byte * period_cnt_ptr;
	byte parse_ctr;
	undefined field4_0xd;
	undefined field5_0xe;
	undefined field6_0xf;
};

struct recipe_pointers {
	byte * recipe;
	byte * layer_len_ptrs[2]; /* Will point at recipe byte 10 */
	undefined field2_0xc;
	undefined field3_0xd;
	undefined field4_0xe;
	undefined field5_0xf;
	struct recipe_layer_fragment layers[2];
	undefined field7_0x30;
	undefined field8_0x31;
	undefined field9_0x32;
	undefined field10_0x33;
	undefined field11_0x34;
	undefined field12_0x35;
	undefined field13_0x36;
	undefined field14_0x37;
	undefined field15_0x38;
	undefined field16_0x39;
	undefined field17_0x3a;
	undefined field18_0x3b;
	undefined field19_0x3c;
	undefined field20_0x3d;
	undefined field21_0x3e;
	undefined field22_0x3f;
	undefined field23_0x40;
	undefined field24_0x41;
	undefined field25_0x42;
	undefined field26_0x43;
};

struct recipe_pointers recipe_pointers;

void Recipe_bytes_to_layers(byte *recipe) {
	byte *pbVar1;
	int iVar2;
	byte *puVar3;
	uint i;
	uint uVar3;
	byte bVar4;
	byte bVar5;
  
	for (i = 0; i < 3; i = (i + 1) & 0xff) {
		recipe_pointers.layers[i].field6_0xf = 0;
		recipe_pointers.layers[i].field5_0xe = 0;
		recipe_pointers.layers[i].field4_0xd = 0;
		recipe_pointers.layers[i].parse_ctr = 0;
	}
	bVar4 = 0;
	puVar3 = (byte *)(recipe + 10);
	recipe_pointers.recipe = recipe;
	/* Loop over layer count. Sample: for(i=0; i<=2; i++) */
	for (i = 0; i <= recipe[8]; i = (i + 1) & 0xff) {
		/* Set layer len?. In sample recipes, this is 02 and 01. */
		recipe_pointers.layer_len_ptrs[i] = puVar3;
		if (*(byte *)(uint32_t *)puVar3 != 0) {
			bVar4 += 1;
		}
		puVar3 = (byte *)((int)(uint32_t *)puVar3 + 1);
	}
	i = 0;
	bVar5 = 0;
	/* For sample recipes this is for(i=0; i<2; i++) */
	while (bVar5 < bVar4) {
		pbVar1 = recipe_pointers.layer_len_ptrs[i];
		if (*pbVar1 != 0) {
			recipe_pointers.layers[i].period_cnt_ptr = puVar3;
			puVar3 = (byte *)((int)(uint32_t *)puVar3 + (uint)*pbVar1 * 2);
			bVar5 += 1;
		}
		i = (i + 1) & 0xff;
	}
	i = 0;
	bVar5 = 0;
	while (bVar5 < bVar4) {
		if (*recipe_pointers.layer_len_ptrs[i] != 0) {
			if (i < 2) {
				recipe_pointers.layers[i].layer_period = (struct recipe_layer_period *)puVar3;
				recipe_pointers.layers[i].appliance_period = (struct recipe_appliance_period *)0x0;
			}
			else {
				recipe_pointers.layers[i].appliance_period = (struct recipe_appliance_period *)puVar3;
				recipe_pointers.layers[i].layer_period = (struct recipe_layer_period *)0x0;
			}
			recipe_pointers.layers[i].parse_ctr = 0;
			while (uVar3 = (uint)recipe_pointers.layers[i].parse_ctr,
			       uVar3 < *recipe_pointers.layer_len_ptrs[i]) {
				if (i < 2) {
					iVar2 = (uint)*recipe_pointers.layers[i].period_cnt_ptr * 0xe;
				}
				else {
					iVar2 = (uint)*recipe_pointers.layers[i].period_cnt_ptr << 2;
				}
				puVar3 = (byte *)((int)(uint32_t *)puVar3 + iVar2);
				recipe_pointers.layers[i].period_cnt_ptr = recipe_pointers.layers[i].period_cnt_ptr + 2;
				recipe_pointers.layers[i].parse_ctr = recipe_pointers.layers[i].parse_ctr + 1;
			}
			recipe_pointers.layers[i].period_cnt_ptr =
				recipe_pointers.layers[i].period_cnt_ptr + uVar3 * -2;
			recipe_pointers.layers[i].parse_ctr = 0;
			bVar5 += 1;
		}
		i = (i + 1) & 0xff;
	}
	return;
}

byte * Recipe_get_current_recipe(void) {
	return recipe_pointers.recipe;
}

bool Recipe_layer_has_periods(uint idx) {
	byte bVar1;
  
	bVar1 = *recipe_pointers.layer_len_ptrs[idx];
	if (bVar1 != 0) {
		bVar1 = 1;
	}
	return (bool)bVar1;
}

bool Recipe_loop_to_next_period(enum Layer l) {
	bool bVar1;
	byte *period_cnt_ptr;
	uint uVar2;
  
	bVar1 = Recipe_layer_has_periods(l);
	if (bVar1) {
		period_cnt_ptr = recipe_pointers.layers[l].period_cnt_ptr;
		uVar2 = (uint)(byte)recipe_pointers.layers[l].field4_0xd;
		if (uVar2 + 1 < (uint)*period_cnt_ptr) {
			if (l < APPLIANCE) {
				recipe_pointers.layers[l].layer_period = recipe_pointers.layers[l].layer_period + 1;
			}
			else {
				recipe_pointers.layers[l].appliance_period = recipe_pointers.layers[l].appliance_period + 1;
			}
			recipe_pointers.layers[l].field4_0xd = recipe_pointers.layers[l].field4_0xd + '\x01';
			bVar1 = false;
		}
		else if ((byte)recipe_pointers.layers[l].field5_0xe + 1 < (uint)period_cnt_ptr[1]) {
			if (l < APPLIANCE) {
				recipe_pointers.layers[l].layer_period = recipe_pointers.layers[l].layer_period + -uVar2;
			}
			else {
				recipe_pointers.layers[l].appliance_period =
					recipe_pointers.layers[l].appliance_period + -uVar2;
			}
			bVar1 = false;
			recipe_pointers.layers[l].field4_0xd = 0;
			recipe_pointers.layers[l].field5_0xe = recipe_pointers.layers[l].field5_0xe + '\x01';
		}
		else if (recipe_pointers.layers[l].parse_ctr + 1 < (uint)*recipe_pointers.layer_len_ptrs[l]) {
			recipe_pointers.layers[l].period_cnt_ptr = period_cnt_ptr + 2;
			if (l < APPLIANCE) {
				recipe_pointers.layers[l].layer_period = recipe_pointers.layers[l].layer_period + 1;
			}
			else {
				recipe_pointers.layers[l].appliance_period = recipe_pointers.layers[l].appliance_period + 1;
			}
			bVar1 = false;
			recipe_pointers.layers[l].field5_0xe = 0;
			recipe_pointers.layers[l].field4_0xd = 0;
			recipe_pointers.layers[l].parse_ctr = recipe_pointers.layers[l].parse_ctr + 1;
		}
		else {
			if (l < APPLIANCE) {
				recipe_pointers.layers[l].layer_period = recipe_pointers.layers[l].layer_period + -uVar2;
			}
			else {
				recipe_pointers.layers[l].appliance_period =
					recipe_pointers.layers[l].appliance_period + -uVar2;
			}
			recipe_pointers.layers[l].field5_0xe = 0;
			recipe_pointers.layers[l].field4_0xd = 0;
			recipe_pointers.layers[l].field6_0xf = 1;
		}
	}
	return bVar1;
}

uint32_t Recipe_parse_timer_seconds(enum Layer l,int total_offset_plus_time) {
	int uVar1;
	uint32_t uVar2;
	uint32_t it=0;
  
	uVar1 = (total_offset_plus_time - *(int *)(recipe_pointers.recipe + 4)) + -86400;
	printf("cycle_start - offset_time = %d\n", uVar1);
	if (l < APPLIANCE) {
		uVar2 = (recipe_pointers.layers[l].layer_period)->length_seconds;
	}
	else {
		uVar2 = (recipe_pointers.layers[l].appliance_period)->length_seconds;
	}
	if (-1 < uVar1) {
		for (; 0 < uVar1; uVar1 -= uVar2) {
			if (l < APPLIANCE) {
				uVar2 = (recipe_pointers.layers[l].layer_period)->length_seconds;
			}
			else {
				uVar2 = (recipe_pointers.layers[l].appliance_period)->length_seconds;
			}
			if (uVar2 < (uint)uVar1) {
				Recipe_loop_to_next_period(l);
			}
			//printf("uVar1 %d, uVar2 %d, layer_period %p\n", uVar1, uVar2, (void*)recipe_pointers.layers[l].layer_period);
			it++;
		}
		printf("Found period after %u iterations\n", it);
		if (uVar1 != 0) {
			uVar2 = -uVar1;
		}
	}
	return uVar2;
}

int Recipe_get_saved_total_offset_plus_current_time(void) {
	uint32_t uVar1;
	time_t tVar2;

	// This is the equivalent of time(2)
	//tVar2 = RTC_get_unix_time();

	struct tm cal = {0};
	cal.tm_sec=0;
	cal.tm_min=15;
	cal.tm_hour=7;
	cal.tm_mday=2;
	//cal.tm_mon=2; //March
	//cal.tm_mon=4;
	//cal.tm_mon=6; //July
	cal.tm_mon=8; // September
	cal.tm_year=2023-1900;
	cal.tm_isdst=1;
		
	tVar2 = mktime(&cal);
	printf("Using timestamp %ld\n", tVar2);

	// This is the value of total_offset. For our purposes, we'll
	// use 68400, which is the equivalent of a 7:00am sunrise in
	// Europe/Berlin
	//uVar1 = RTC_get_backup_value(1);
	uVar1 = 68400;
	
	return uVar1 + (int)tVar2;
}

int main(void) {
	if (sizeof(struct recipe_layer_period)!=0xe) {
		printf("Wrong recipe layer size. Want 0xe, got 0x%x\n", sizeof(struct recipe_layer_period));
		return 1;
	}
	
	Recipe_bytes_to_layers(recipe_gen);
	
	int totalOffsetPlusTime;
	uint *pRecipeId;
	bool bVar7;
	uint32_t uVar10;
	enum Layer layer;
	pRecipeId = (uint *)Recipe_get_current_recipe();
	printf("Recipe ID %d, pointer %p\n",*pRecipeId,(void*)pRecipeId);
	if ((*pRecipeId == 1) || (0xfe < *pRecipeId)) {
		totalOffsetPlusTime = Recipe_get_saved_total_offset_plus_current_time();
	}
	else {
		totalOffsetPlusTime = 0;
	}
	for (layer = LAYER_A; layer < LAYER_MAX; layer = (layer + LAYER_B) & 0xff) {
		bVar7 = Recipe_layer_has_periods(layer);
		if (bVar7) {
			uVar10 = Recipe_parse_timer_seconds(layer,totalOffsetPlusTime);
			printf("layer %d, timer_seconds: %u = %uh%um%us\n", layer, uVar10, uVar10/3600, (uVar10%3600)/60, (uVar10%60));
		} else {
			printf("layer %d, no periods\n", layer);
		}
	}
}
