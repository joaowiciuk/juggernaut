/*
 ============================================================================
 Name        : read_ads1115.c
 Author      : João Wiciuk
 Description : Prints analog values using ADS1115
 ============================================================================
 */

#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <unistd.h>
#include <linux/i2c-dev.h>
#include <sys/ioctl.h>
#include <fcntl.h>

#include "ads1115_rpi.h"

#define N 5

int main(void) {
	if (openI2CBus("/dev/i2c-1") == -1) {
		return EXIT_FAILURE;
	}
	setI2CSlave(0x48);
	int i;
	float s2, sum, sum_of_sqr, cte;
	for (i = 0; i < N; i++) {
		//printf("%.2f\n", readVoltage(0));
		sum += readVoltage(0);
		sum_of_sqr += pow(readVoltage(0), 2); 
	}
	cte = 1 / ((float) N * ((float) N - 1));
	s2 = cte * ((float) N * sum_of_sqr - pow(sum, 2));
	printf("%.2f\n", s2);
	return EXIT_SUCCESS;
}
