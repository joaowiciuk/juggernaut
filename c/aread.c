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
	float x[N];
    float average, variance, std_deviation, sum1 = 0;

	if (openI2CBus("/dev/i2c-1") == -1) {
		return EXIT_FAILURE;
	}
	setI2CSlave(0x48);
	int i;
	float s2, sum, sum_of_sqr, cte;
	for (i = 0; i < N; i++) {
		x[i] = readVoltage(0);
		printf("%.2f\n", x[i]);
		sum += x[i];
		sum_of_sqr += pow(x[i], 2); 
	}

	average = sum / (float) N;
    /*  Compute  variance  and standard deviation  */
    for (i = 0; i < N; i++) {
        sum1 = sum1 + pow((x[i] - average), 2);
    }
    variance = sum1 / (float)N;
    std_deviation = sqrt(variance);
    printf("Average of all elements = %.2f\n", average);
    printf("variance of all elements = %.2f\n", variance);
    printf("Standard deviation = %.2f\n", std_deviation);

	cte = 1 / ((float) N * ((float) N - 1));
	s2 = cte * ((float) N * sum_of_sqr - pow(sum, 2));
	printf("cte: %.2f\n", cte);
	printf("sum: %.2f\n", sum);
	printf("sum_of_sqr: %.2f\n", sum_of_sqr);
	printf("variance: %.2f\n", s2);
	return EXIT_SUCCESS;
}
